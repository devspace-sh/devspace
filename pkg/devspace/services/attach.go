package services

import (
	"context"
	"os"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/mgutz/ansi"
)

// StartAttach opens a new terminal
func (serviceClient *client) StartAttach(options targetselector.Options, interrupt chan error) error {
	options = options.WithQuestion("Which pod do you want to attach to?")
	container, err := targetselector.GlobalTargetSelector.SelectSingleContainer(context.TODO(), serviceClient.client, options, serviceClient.log)
	if err != nil {
		return err
	}

	wrapper, upgradeRoundTripper, err := serviceClient.client.GetUpgraderWrapper()
	if err != nil {
		return err
	}

	if !container.Container.TTY || !container.Container.Stdin {
		serviceClient.log.Warnf("To be able to interact with the container its options tty (currently `%t`) and stdin (currently `%t`) must both be `true`", container.Container.TTY, container.Container.Stdin)
	}

	serviceClient.log.Infof("Attaching to pod:container %s:%s", ansi.Color(container.Pod.Name, "white+b"), ansi.Color(container.Container.Name, "white+b"))
	serviceClient.log.Info("If you don't see a command prompt, try pressing enter.")

	go func() {
		interrupt <- serviceClient.client.ExecStreamWithTransport(&kubectl.ExecStreamWithTransportOptions{
			ExecStreamOptions: kubectl.ExecStreamOptions{
				Pod:       container.Pod,
				Container: container.Container.Name,
				TTY:       container.Container.TTY,
				Stdin:     os.Stdin,
				Stdout:    os.Stdout,
				Stderr:    os.Stderr,
			},
			Transport:   wrapper,
			Upgrader:    upgradeRoundTripper,
			SubResource: kubectl.SubResourceAttach,
		})
	}()

	err = <-interrupt
	upgradeRoundTripper.Close()
	return err
}
