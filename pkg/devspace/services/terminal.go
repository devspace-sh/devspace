package services

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/covexo/devspace/pkg/devspace/config/v1"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/transport/spdy"
	kubectlExec "k8s.io/client-go/util/exec"
)

// StartTerminal opens a new terminal
func StartTerminal(client *kubernetes.Clientset, serviceNameOverride, containerNameOverride, labelSelectorOverride, namespaceOverride string, args []string, interrupt chan error, log log.Logger) error {
	var command []string
	config := configutil.GetConfig()

	if len(args) == 0 && (config.DevSpace.Terminal.Command == nil || len(*config.DevSpace.Terminal.Command) == 0) {
		command = []string{
			"sh",
			"-c",
			"command -v bash >/dev/null 2>&1 && exec bash || exec sh",
		}
	} else {
		if len(args) > 0 {
			command = args
		} else {
			for _, cmd := range *config.DevSpace.Terminal.Command {
				command = append(command, *cmd)
			}
		}
	}

	service, namespace, labelSelector, err := getServiceNamespaceLabelSelector(serviceNameOverride, labelSelectorOverride, namespaceOverride)
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
		if service != nil && service.ContainerName != nil {
			containerName = *service.ContainerName
		} else {
			if config.DevSpace.Terminal.ContainerName != nil {
				containerName = *config.DevSpace.Terminal.ContainerName
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

func getServiceNamespaceLabelSelector(serviceNameOverride, labelSelectorOverride, namespaceOverride string) (*v1.ServiceConfig, string, string, error) {
	config := configutil.GetConfig()

	var service *v1.ServiceConfig
	serviceName := "default"

	if serviceNameOverride == "" {
		if config.DevSpace.Terminal.Service != nil {
			serviceName = *config.DevSpace.Terminal.Service
		}
	} else {
		serviceName = serviceNameOverride
	}

	if serviceName != "" {
		var err error

		service, err = configutil.GetService(serviceName)
		if err != nil && serviceName != "default" {
			return nil, "", "", fmt.Errorf("Error resolving service name: %v", err)
		}
	}

	// Select pods
	namespace := ""
	if namespaceOverride == "" {
		if service != nil && service.Namespace != nil {
			namespace = *service.Namespace
		} else {
			if config.DevSpace.Terminal != nil && config.DevSpace.Terminal.Namespace != nil {
				namespace = *config.DevSpace.Terminal.Namespace
			}
		}
	} else {
		namespace = namespaceOverride
	}

	labelSelector := ""
	// Retrieve pod from label selector
	if labelSelectorOverride == "" {
		labelSelector = "release=" + GetNameOfFirstHelmDeployment()

		if service != nil {
			labels := make([]string, 0, len(*service.LabelSelector)-1)
			for key, value := range *service.LabelSelector {
				labels = append(labels, key+"="+*value)
			}

			labelSelector = strings.Join(labels, ", ")
		} else {
			if config.DevSpace.Terminal != nil && config.DevSpace.Terminal.LabelSelector != nil {
				labels := make([]string, 0, len(*config.DevSpace.Terminal.LabelSelector))
				for key, value := range *config.DevSpace.Terminal.LabelSelector {
					labels = append(labels, key+"="+*value)
				}

				labelSelector = strings.Join(labels, ", ")
			}
		}
	} else {
		labelSelector = labelSelectorOverride
	}

	return service, namespace, labelSelector, nil
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

	if config.DevSpace.Deployments != nil {
		for _, deploymentConfig := range *config.DevSpace.Deployments {
			if deploymentConfig.Helm != nil {
				return *deploymentConfig.Name
			}
		}
	}

	return configutil.DefaultDevspaceDeploymentName
}
