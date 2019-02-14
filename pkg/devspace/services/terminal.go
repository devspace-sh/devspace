package services

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	v1 "github.com/covexo/devspace/pkg/devspace/config/versions/latest"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/transport/spdy"
	kubectlExec "k8s.io/client-go/util/exec"
)

// StartTerminal opens a new terminal
func StartTerminal(client *kubernetes.Clientset, selectorNameOverride, containerNameOverride, labelSelectorOverride, namespaceOverride string, args []string, interrupt chan error, log log.Logger) error {
	var command []string
	config := configutil.GetConfig()

	if len(args) == 0 && (config.Dev.Terminal.Command == nil || len(*config.Dev.Terminal.Command) == 0) {
		command = []string{
			"sh",
			"-c",
			"command -v bash >/dev/null 2>&1 && exec bash || exec sh",
		}
	} else {
		if len(args) > 0 {
			command = args
		} else {
			for _, cmd := range *config.Dev.Terminal.Command {
				command = append(command, *cmd)
			}
		}
	}

	selector, namespace, labelSelector, err := getSelectorNamespaceLabelSelector(selectorNameOverride, labelSelectorOverride, namespaceOverride)
	if err != nil {
		return err
	}

	// Get first running pod
	log.StartWait("Terminal: Waiting for pods...")
	pod, err := kubectl.GetNewestRunningPod(client, labelSelector, namespace, time.Second*120)
	log.StopWait()
	if err != nil {
		return fmt.Errorf("Error starting terminal: Cannot find running pod: %v", err)
	}

	// Get container name
	containerName := pod.Spec.Containers[0].Name
	if containerNameOverride == "" {
		if selector != nil && selector.ContainerName != nil {
			containerName = *selector.ContainerName
		} else {
			if config.Dev.Terminal.ContainerName != nil {
				containerName = *config.Dev.Terminal.ContainerName
			}
		}
	} else {
		containerName = containerNameOverride
	}

	wrapper, upgradeRoundTripper, err := getUpgraderWrapper()
	if err != nil {
		return err
	}

	go func() {
		terminalErr := kubectl.ExecStreamWithTransport(wrapper, upgradeRoundTripper, client, pod, containerName, command, true, os.Stdin, os.Stdout, os.Stderr)
		if terminalErr != nil {
			if _, ok := terminalErr.(kubectlExec.CodeExitError); ok == false {
				interrupt <- fmt.Errorf("Unable to start terminal session: %v", terminalErr)
				return
			}
		}

		interrupt <- nil
	}()

	err = <-interrupt
	upgradeRoundTripper.Close()
	return err
}

func getSelectorNamespaceLabelSelector(serviceNameOverride, labelSelectorOverride, namespaceOverride string) (*v1.SelectorConfig, string, string, error) {
	config := configutil.GetConfig()

	var selector *v1.SelectorConfig
	selectorName := "default"

	if serviceNameOverride == "" {
		if config.Dev.Terminal != nil && config.Dev.Terminal.Selector != nil {
			selectorName = *config.Dev.Terminal.Selector
		}
	} else {
		selectorName = serviceNameOverride
	}

	if selectorName != "" {
		var err error

		selector, err = configutil.GetSelector(selectorName)
		if err != nil && selectorName != "default" {
			return nil, "", "", fmt.Errorf("Error resolving service name: %v", err)
		}
	}

	// Select pods
	namespace := ""
	if namespaceOverride == "" {
		if selector != nil && selector.Namespace != nil {
			namespace = *selector.Namespace
		} else {
			if config.Dev.Terminal != nil && config.Dev.Terminal.Namespace != nil {
				namespace = *config.Dev.Terminal.Namespace
			}
		}
	} else {
		namespace = namespaceOverride
	}

	labelSelector := ""
	// Retrieve pod from label selector
	if labelSelectorOverride == "" {
		labelSelector = "release=" + GetNameOfFirstHelmDeployment()

		if selector != nil {
			labels := make([]string, 0, len(*selector.LabelSelector)-1)
			for key, value := range *selector.LabelSelector {
				labels = append(labels, key+"="+*value)
			}

			labelSelector = strings.Join(labels, ", ")
		} else {
			if config.Dev.Terminal != nil && config.Dev.Terminal.LabelSelector != nil {
				labels := make([]string, 0, len(*config.Dev.Terminal.LabelSelector))
				for key, value := range *config.Dev.Terminal.LabelSelector {
					labels = append(labels, key+"="+*value)
				}

				labelSelector = strings.Join(labels, ", ")
			}
		}
	} else {
		labelSelector = labelSelectorOverride
	}

	return selector, namespace, labelSelector, nil
}

type upgraderWrapper struct {
	Upgrader    spdy.Upgrader
	Connections []httpstream.Connection
}

func (uw *upgraderWrapper) NewConnection(resp *http.Response) (httpstream.Connection, error) {
	conn, err := uw.Upgrader.NewConnection(resp)
	if err != nil {
		return nil, err
	}

	uw.Connections = append(uw.Connections, conn)

	return conn, nil
}

func (uw *upgraderWrapper) Close() error {
	for _, conn := range uw.Connections {
		err := conn.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func getUpgraderWrapper() (http.RoundTripper, *upgraderWrapper, error) {
	kubeconfig, err := kubectl.GetClientConfig()
	if err != nil {
		return nil, nil, err
	}

	wrapper, upgradeRoundTripper, err := spdy.RoundTripperFor(kubeconfig)
	if err != nil {
		return nil, nil, err
	}

	return wrapper, &upgraderWrapper{
		Upgrader:    upgradeRoundTripper,
		Connections: make([]httpstream.Connection, 0, 1),
	}, nil
}

// GetNameOfFirstHelmDeployment retrieves the first helm deployment name
func GetNameOfFirstHelmDeployment() string {
	config := configutil.GetConfig()

	if config.Deployments != nil {
		for _, deploymentConfig := range *config.Deployments {
			if deploymentConfig.Helm != nil {
				return *deploymentConfig.Name
			}
		}
	}

	return configutil.DefaultDevspaceDeploymentName
}
