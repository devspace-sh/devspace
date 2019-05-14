package services

import (
	"fmt"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"k8s.io/client-go/kubernetes"
	kubectlExec "k8s.io/client-go/util/exec"
)

// StartAttach starts attaching to the first pod devspace finds or does nothing
func StartAttach(config *latest.Config, client kubernetes.Interface, cmdParameter targetselector.CmdParameter, interrupt chan error, log log.Logger) error {
	selectorParameter := &targetselector.SelectorParameter{
		CmdParameter: cmdParameter,
	}

	if config != nil && config.Dev != nil && config.Dev.Terminal != nil {
		selectorParameter.ConfigParameter = targetselector.ConfigParameter{
			Selector:      config.Dev.Terminal.Selector,
			Namespace:     config.Dev.Terminal.Namespace,
			LabelSelector: config.Dev.Terminal.LabelSelector,
			ContainerName: config.Dev.Terminal.ContainerName,
		}
	}

	targetSelector, err := targetselector.NewTargetSelector(config, selectorParameter, true)
	if err != nil {
		return err
	}

	pod, container, err := targetSelector.GetContainer(client)
	if err != nil {
		return err
	}

	kubeconfig, err := kubectl.GetClientConfig(config)
	if err != nil {
		return err
	}

	wrapper, upgradeRoundTripper, err := kubectl.GetUpgraderWrapper(kubeconfig)
	if err != nil {
		return err
	}

	log.Infof("Printing logs of pod %s/%s...", pod.Name, container.Name)

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
