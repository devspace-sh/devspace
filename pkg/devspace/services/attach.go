package services

import (
	"fmt"
	"os"
	"time"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"k8s.io/client-go/kubernetes"
	kubectlExec "k8s.io/client-go/util/exec"
)

// StartAttach starts attaching to the first pod devspace finds or does nothing
func StartAttach(client *kubernetes.Clientset, selectorNameOverride, containerNameOverride, labelSelectorOverride, namespaceOverride string, interrupt chan error, log log.Logger) error {
	config := configutil.GetConfig()

	selector, namespace, labelSelector, err := getSelectorNamespaceLabelSelector(selectorNameOverride, labelSelectorOverride, namespaceOverride)
	if err != nil {
		return err
	}

	pod, err := kubectl.GetNewestRunningPod(client, labelSelector, namespace, time.Second*10)
	if err != nil {
		return fmt.Errorf("Cannot find running pod: %v", err)
	}

	// Get container name
	containerName := pod.Spec.Containers[0].Name
	if containerNameOverride == "" {
		if selector != nil && selector.ContainerName != nil {
			containerName = *selector.ContainerName
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

	log.Infof("Printing logs of pod %s/%s...", pod.Name, containerName)

	go func() {
		err := kubectl.AttachStreamWithTransport(wrapper, upgradeRoundTripper, client, pod, containerName, true, nil, os.Stdout, os.Stderr)
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
