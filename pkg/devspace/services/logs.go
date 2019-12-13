package services

import (
	"context"
	"io"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/mgutz/ansi"
)

// StartLogs print the logs and then attaches to the container
func (serviceClient *client) StartLogs(follow bool, tail int64) error {
	return serviceClient.StartLogsWithWriter(follow, tail, os.Stdout)
}

// StartLogsWithWriter prints the logs and then attaches to the container with the given stdout and stderr
func (serviceClient *client) StartLogsWithWriter(follow bool, tail int64, writer io.Writer) error {
	targetSelector, err := targetselector.NewTargetSelector(serviceClient.config, serviceClient.client, serviceClient.selectorParameter, true, nil)
	if err != nil {
		return err
	}

	// Allow picking non running pods
	targetSelector.AllowNonRunning = true

	pod, container, err := targetSelector.GetContainer(true, serviceClient.log)
	if err != nil {
		return err
	}

	serviceClient.log.Infof("Printing logs of pod:container %s:%s", ansi.Color(pod.Name, "white+b"), ansi.Color(container.Name, "white+b"))

	reader, err := serviceClient.client.Logs(context.Background(), pod.Namespace, pod.Name, container.Name, false, &tail, follow)
	if err != nil {
		return nil
	}

	_, err = io.Copy(writer, reader)
	return err
}
