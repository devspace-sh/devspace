package services

import (
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"github.com/mgutz/ansi"
)

// StartAttach opens a new terminal
func (serviceClient *client) StartAttach(imageSelector []string, interrupt chan error) error {
	targetSelector, err := targetselector.NewTargetSelector(serviceClient.config, serviceClient.client, serviceClient.selectorParameter, true, imageSelector)
	if err != nil {
		return err
	}

	targetSelector.PodQuestion = ptr.String("Which pod do you want to attach to?")

	pod, container, err := targetSelector.GetContainer(true, serviceClient.log)
	if err != nil {
		return err
	}

	wrapper, upgradeRoundTripper, err := serviceClient.client.GetUpgraderWrapper()
	if err != nil {
		return err
	}

	if container.TTY == false || container.Stdin == false {
		serviceClient.log.Warnf("To be able to interact with the container its options tty (currently `%t`) and stdin (currently `%t`) must both be `true`", container.TTY, container.Stdin)
	}

	serviceClient.log.Infof("Attaching to pod:container %s:%s", ansi.Color(pod.Name, "white+b"), ansi.Color(container.Name, "white+b"))
	serviceClient.log.Info("If you don't see a command prompt, try pressing enter.")

	go func() {
		interrupt <- serviceClient.client.ExecStreamWithTransport(wrapper, upgradeRoundTripper, pod, container.Name, nil, container.TTY, os.Stdin, os.Stdout, os.Stderr, kubectl.SubResourceAttach)
	}()

	err = <-interrupt
	upgradeRoundTripper.Close()
	return err
}
