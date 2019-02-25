package services

import (
	"fmt"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	kubectlExec "k8s.io/client-go/util/exec"
)

// StartLogs print the logs and then attaches to the container
func StartLogs(client *kubernetes.Clientset, selectorNameOverride, containerNameOverride, labelSelectorOverride, namespaceOverride string, pick, follow bool, tail int64, log log.Logger) error {
	config := configutil.GetConfig()

	selector, namespace, labelSelector, err := getSelectorNamespaceLabelSelector(selectorNameOverride, labelSelectorOverride, namespaceOverride)
	if err != nil {
		return err
	}

	// Get container name
	var containerName *string
	if containerNameOverride == "" {
		if selector != nil && selector.ContainerName != nil {
			containerName = selector.ContainerName
		} else {
			if config.Dev != nil && config.Dev.Terminal != nil && config.Dev.Terminal.ContainerName != nil {
				containerName = config.Dev.Terminal.ContainerName
			}
		}
	} else {
		containerName = &containerNameOverride
	}

	var (
		pod       *v1.Pod
		container *v1.Container
	)

	if pick {
		pod, container, err = SelectContainer(client, namespace, nil, nil)
		if err != nil {
			return err
		}
		if pod == nil || container == nil {
			return fmt.Errorf("No pod found")
		}
	} else {
		pod, container, err = SelectContainer(client, namespace, &labelSelector, containerName)
		if err != nil {
			return err
		}
		if pod == nil || container == nil {
			return fmt.Errorf("No pod found")
		}
	}

	kubeconfig, err := kubectl.GetClientConfig()
	if err != nil {
		return err
	}

	wrapper, upgradeRoundTripper, err := kubectl.GetUpgraderWrapper(kubeconfig)
	if err != nil {
		return err
	}

	log.Infof("Printing logs of pod:container %s:%s", ansi.Color(pod.Name, "white+b"), ansi.Color(container.Name, "white+b"))

	logOutput, err := kubectl.Logs(client, namespace, pod.Name, container.Name, false, &tail)
	if err != nil {
		return nil
	}

	log.WriteString(logOutput)
	if follow == false {
		if logOutput == "" {
			log.Infof("Logs of pod %s:%s were empty", ansi.Color(pod.Name, "white+b"), ansi.Color(container.Name, "white+b"))
		}

		return nil
	}

	interrupt := make(chan error)

	go func() {
		err := kubectl.AttachStreamWithTransport(wrapper, upgradeRoundTripper, client, pod, container.Name, true, nil, os.Stdout, os.Stderr)
		if err != nil {
			if _, ok := err.(kubectlExec.CodeExitError); ok == false {
				interrupt <- fmt.Errorf("Unable to start attach session: %v", err)
				return
			}
		}

		interrupt <- nil
	}()

	err = <-interrupt
	upgradeRoundTripper.Close()

	return err
}
