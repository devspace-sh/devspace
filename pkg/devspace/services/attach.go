package services

import (
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"github.com/mgutz/ansi"
)

// StartAttach opens a new terminal
func StartAttach(config *latest.Config, client *kubectl.Client, selectorParameter *targetselector.SelectorParameter, imageSelector []string, interrupt chan error, log log.Logger) error {
	targetSelector, err := targetselector.NewTargetSelector(config, client, selectorParameter, true, imageSelector)
	if err != nil {
		return err
	}

	targetSelector.PodQuestion = ptr.String("Which pod do you want to attach to?")

	pod, container, err := targetSelector.GetContainer(log)
	if err != nil {
		return err
	}

	wrapper, upgradeRoundTripper, err := kubectl.GetUpgraderWrapper(client.RestConfig)
	if err != nil {
		return err
	}

	log.Infof("Attaching to pod:container %s:%s", ansi.Color(pod.Name, "white+b"), ansi.Color(container.Name, "white+b"))

	go func() {
		interrupt <- client.ExecStreamWithTransport(wrapper, upgradeRoundTripper, pod, container.Name, nil, true, os.Stdin, os.Stdout, os.Stderr, kubectl.SubResourceAttach)
	}()

	err = <-interrupt
	upgradeRoundTripper.Close()
	return err
}
