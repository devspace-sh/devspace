package services

import (
	"context"
	"io"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
)

// StartLogs print the logs and then attaches to the container
func StartLogs(config *latest.Config, client kubectl.Client, cmdParameter targetselector.CmdParameter, follow bool, tail int64, log log.Logger) error {
	return StartLogsWithWriter(config, client, cmdParameter, follow, tail, log, os.Stdout)
}

// StartLogsWithWriter prints the logs and then attaches to the container with the given stdout and stderr
func StartLogsWithWriter(config *latest.Config, client kubectl.Client, cmdParameter targetselector.CmdParameter, follow bool, tail int64, log log.Logger, writer io.Writer) error {
	selectorParameter := &targetselector.SelectorParameter{
		CmdParameter: cmdParameter,
	}

	targetSelector, err := targetselector.NewTargetSelector(config, client, selectorParameter, true, nil)
	if err != nil {
		return err
	}

	// Allow picking non running pods
	targetSelector.AllowNonRunning = true

	pod, container, err := targetSelector.GetContainer(true, log)
	if err != nil {
		return err
	}

	log.Infof("Printing logs of pod:container %s:%s", ansi.Color(pod.Name, "white+b"), ansi.Color(container.Name, "white+b"))

	reader, err := client.Logs(context.Background(), pod.Namespace, pod.Name, container.Name, false, &tail, follow)
	if err != nil {
		return nil
	}

	_, err = io.Copy(writer, reader)
	return err
}
