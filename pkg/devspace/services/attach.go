package services

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	v1 "github.com/covexo/devspace/pkg/devspace/config/versions/latest"
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

	pod, err := kubectl.GetNewestRunningPod(client, labelSelector, namespace, time.Second*60)
	if err != nil {
		return err
	}

	// Get container name
	container := pod.Spec.Containers[0].Name
	if containerNameOverride == "" {
		if selector != nil && selector.ContainerName != nil {
			container = *selector.ContainerName
		} else {
			if config.Dev != nil && config.Dev.Terminal != nil && config.Dev.Terminal.ContainerName != nil {
				container = *config.Dev.Terminal.ContainerName
			}
		}
	} else {
		container = containerNameOverride
	}

	kubeconfig, err := kubectl.GetClientConfig()
	if err != nil {
		return err
	}

	wrapper, upgradeRoundTripper, err := kubectl.GetUpgraderWrapper(kubeconfig)
	if err != nil {
		return err
	}

	log.Infof("Printing logs of pod %s/%s...", pod.Name, container)

	go func() {
		err := kubectl.AttachStreamWithTransport(wrapper, upgradeRoundTripper, client, pod, container, true, nil, os.Stdout, os.Stderr)
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
	namespace, err := configutil.GetDefaultNamespace(config)
	if err != nil {
		return nil, "", "", err
	}

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
