package kubectl

import (
	"context"
	"github.com/pkg/errors"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

// ReadLogs reads the logs and returns a string
func (client *client) ReadLogs(ctx context.Context, namespace, podName, containerName string, lastContainerLog bool, tail *int64) (string, error) {
	readCloser, err := client.Logs(ctx, namespace, podName, containerName, lastContainerLog, tail, false)
	if err != nil {
		return "", err
	}

	logs, err := io.ReadAll(readCloser)
	if err != nil {
		return "", err
	}

	return string(logs), nil
}

// Logs prints the container logs
func (client *client) Logs(ctx context.Context, namespace, podName, containerName string, lastContainerLog bool, tail *int64, follow bool) (io.ReadCloser, error) {
	lines := int64(500)
	if tail != nil {
		lines = *tail
	}

	request := client.KubeClient().CoreV1().RESTClient().Get().Namespace(namespace).Name(podName).Resource("pods").SubResource("log").VersionedParams(&v1.PodLogOptions{
		Container: containerName,
		TailLines: &lines,
		Previous:  lastContainerLog,
		Follow:    follow,
	}, scheme.ParameterCodec)
	if request.URL().String() == "" {
		return nil, errors.New("Request url is empty")
	}

	return request.Stream(ctx)
}
