package services

import (
	"fmt"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	kubectlExec "k8s.io/client-go/util/exec"
)

// StartAttach starts attaching to the first pod devspace finds or does nothing
func StartAttach(config *latest.Config, client *kubectl.Client, cmdParameter targetselector.CmdParameter, interrupt chan error, log log.Logger) (int, error) {
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

	targetSelector, err := targetselector.NewTargetSelector(config, client, selectorParameter, true)
	if err != nil {
		return 0, err
	}

	pod, container, err := targetSelector.GetContainer()
	if err != nil {
		return 0, err
	}

	wrapper, upgradeRoundTripper, err := kubectl.GetUpgraderWrapper(client.RestConfig)
	if err != nil {
		return 0, err
	}

	log.Infof("Printing logs of pod %s/%s...", pod.Name, container.Name)

	go func() {
		interrupt <- client.AttachStreamWithTransport(wrapper, upgradeRoundTripper, pod, container.Name, true, nil, os.Stdout, os.Stderr)
	}()

	err = <-interrupt
	upgradeRoundTripper.Close()
	if err != nil {
		if exitError, ok := err.(kubectlExec.CodeExitError); ok {
			return exitError.Code, nil
		}

		return 0, fmt.Errorf("Unable to start terminal session: %v", err)
	}

	return 0, nil
}
