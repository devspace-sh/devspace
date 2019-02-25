package services

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	kubectlExec "k8s.io/client-go/util/exec"
)

// StartTerminal opens a new terminal
func StartTerminal(client *kubernetes.Clientset, selectorNameOverride, containerNameOverride, labelSelectorOverride, namespaceOverride string, pick bool, args []string, interrupt chan error, log log.Logger) error {
	var command []string
	config := configutil.GetConfig()

	customCommand := false
	if config.Dev != nil && config.Dev.Terminal != nil && config.Dev.Terminal.Command != nil && len(*config.Dev.Terminal.Command) > 0 {
		customCommand = true
	}

	if len(args) == 0 && customCommand == false {
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

	var pod *v1.Pod
	var container *v1.Container
	var containerName string

	if pick {
		pod, container, err = SelectContainer(client, namespace, nil, nil)
		if err != nil {
			return err
		}
		if pod == nil || container == nil {
			return fmt.Errorf("No pod found")
		}

		containerName = container.Name
	} else {
		// Get first running pod
		log.StartWait("Terminal: Waiting for pods...")
		pod, err = kubectl.GetNewestRunningPod(client, labelSelector, namespace, time.Second*5)
		log.StopWait()
		if err != nil {
			pod, container, err = SelectContainer(client, namespace, nil, nil)
			if err != nil {
				return fmt.Errorf("Error starting terminal: Cannot find running pod: %v", err)
			}
			if pod == nil || container == nil {
				return errors.New("Error starting terminal: Cannot find a running pod")
			}

			containerName = container.Name
		} else {
			// Get container name
			containerName = pod.Spec.Containers[0].Name
			if containerNameOverride == "" {
				if selector != nil && selector.ContainerName != nil {
					containerName = *selector.ContainerName
				} else {
					if config.Dev != nil && config.Dev.Terminal != nil && config.Dev.Terminal.ContainerName != nil {
						containerName = *config.Dev.Terminal.ContainerName
					}
				}
			} else {
				containerName = containerNameOverride
			}
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
