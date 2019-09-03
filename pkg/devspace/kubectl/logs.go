package kubectl

import (
	"context"
	"io/ioutil"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

// Logs prints the container logs
func (client *Client) Logs(namespace, podName, containerName string, lastContainerLog bool, tail *int64) (string, error) {
	lines := int64(100)
	if tail != nil {
		lines = *tail
	}

	request := client.Client.CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{
		Container: containerName,
		TailLines: &lines,
		Previous:  lastContainerLog,
	})

	if request.URL().String() == "" {
		return "", errors.New("Request url is empty")
	}

	reader, err := request.Context(context.Background()).Stream()
	if err != nil {
		return "", err
	}

	logs, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(logs), nil
}
