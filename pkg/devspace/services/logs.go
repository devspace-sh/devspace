package services

import (
	"fmt"
	"io"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	kubectlExec "k8s.io/client-go/util/exec"
)

// StartLogs print the logs and then attaches to the container
func StartLogs(config *latest.Config, client *kubectl.Client, cmdParameter targetselector.CmdParameter, follow bool, tail int64, log log.Logger) error {
	return StartLogsWithWriter(config, client, cmdParameter, follow, tail, log, os.Stdout, os.Stderr)
}

// StartLogsWithWriter prints the logs and then attaches to the container with the given stdout and stderr
func StartLogsWithWriter(config *latest.Config, client *kubectl.Client, cmdParameter targetselector.CmdParameter, follow bool, tail int64, log log.Logger, stdout io.Writer, stderr io.Writer) error {
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
		return err
	}

	pod, container, err := targetSelector.GetContainer()
	if err != nil {
		return err
	}

	wrapper, upgradeRoundTripper, err := kubectl.GetUpgraderWrapper(client.RestConfig)
	if err != nil {
		return err
	}

	log.Infof("Printing logs of pod:container %s:%s", ansi.Color(pod.Name, "white+b"), ansi.Color(container.Name, "white+b"))

	logOutput, err := client.Logs(pod.Namespace, pod.Name, container.Name, false, &tail)
	if err != nil {
		return nil
	}

	stdout.Write([]byte(logOutput))
	if follow == false {
		if logOutput == "" {
			log.Infof("Logs of pod %s:%s were empty", ansi.Color(pod.Name, "white+b"), ansi.Color(container.Name, "white+b"))
		}

		return nil
	}

	interrupt := make(chan error)

	// TODO: Refactor this, because with this method we could miss some messages between logs and attach
	go func() {
		err := client.AttachStreamWithTransport(wrapper, upgradeRoundTripper, pod, container.Name, true, nil, stdout, stderr)
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
