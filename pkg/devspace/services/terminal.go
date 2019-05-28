package services

import (
	"fmt"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"github.com/mgutz/ansi"
	"k8s.io/client-go/kubernetes"
	kubectlExec "k8s.io/client-go/util/exec"
)

// StartTerminal opens a new terminal
func StartTerminal(config *latest.Config, client kubernetes.Interface, cmdParameter targetselector.CmdParameter, args []string, interrupt chan error, log log.Logger) error {
	command := getCommand(config, args)

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

	targetSelector.PodQuestion = ptr.String("Which pod do you want to open the terminal for?")

	pod, container, err := targetSelector.GetContainer(client)
	if err != nil {
		return err
	}

	kubeconfig, err := kubectl.GetRestConfig(config)
	if err != nil {
		return err
	}

	wrapper, upgradeRoundTripper, err := kubectl.GetUpgraderWrapper(kubeconfig)
	if err != nil {
		return err
	}

	log.Infof("Opening shell to pod:container %s:%s", ansi.Color(pod.Name, "white+b"), ansi.Color(container.Name, "white+b"))

	go func() {
		terminalErr := kubectl.ExecStreamWithTransport(wrapper, upgradeRoundTripper, client, pod, container.Name, command, true, os.Stdin, os.Stdout, os.Stderr)
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

func getCommand(config *latest.Config, args []string) []string {
	var command []string

	if config != nil && config.Dev != nil && config.Dev.Terminal != nil && config.Dev.Terminal.Command != nil && len(*config.Dev.Terminal.Command) > 0 {
		for _, cmd := range *config.Dev.Terminal.Command {
			command = append(command, *cmd)
		}
	}

	if len(args) > 0 {
		command = args
	} else {
		if len(command) == 0 {
			command = []string{
				"sh",
				"-c",
				"command -v bash >/dev/null 2>&1 && exec bash || exec sh",
			}
		}
	}

	return command
}
