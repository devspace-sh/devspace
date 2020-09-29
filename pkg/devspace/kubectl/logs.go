package kubectl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"time"

	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/mgutz/ansi"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

const k8sComponentLabel = "app.kubernetes.io/component"

// ReadLogs reads the logs and returns a string
func (client *client) ReadLogs(namespace, podName, containerName string, lastContainerLog bool, tail *int64) (string, error) {
	readCloser, err := client.Logs(context.Background(), namespace, podName, containerName, lastContainerLog, tail, false)
	if err != nil {
		return "", err
	}

	logs, err := ioutil.ReadAll(readCloser)
	if err != nil {
		return "", err
	}

	return string(logs), nil
}

type logLine struct {
	line  string
	name  string
	color string
}

// LogMultipleTimeout will log multiple and wait for a specific time for ready pods until timeout
func (client *client) LogMultipleTimeout(imageSelector []string, interrupt chan error, tail *int64, writer io.Writer, timeout time.Duration, log logpkg.Logger) error {
	// Get pods
	log.StartWait("Find running pods...")
	pods, err := client.GetRunningPodsWithImage(imageSelector, client.namespace, timeout)
	log.StopWait()
	if err != nil {
		return errors.Errorf("Error finding images: %v", err)
	}
	if len(pods) == 0 {
		return nil
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	if tail == nil {
		tail = ptr.Int64(100)
	}

	defer cancelFunc()

	// Make channel buffered
	lines := make(chan *logLine, 100)
	done := make(chan bool)

	var wg sync.WaitGroup

	printInfo := true

	// Loop over pods and open logs connection
	for idx, pod := range pods {
	Outer:
		for _, container := range pod.Spec.Containers {
			for _, imageName := range imageSelector {
				if compareImageNames(imageName, container.Image) {
					reader, err := client.Logs(ctx, pod.Namespace, pod.Name, container.Name, false, tail, true)
					if err != nil {
						log.Warnf("Couldn't log %s/%s: %v", pod.Name, container.Name, err)
						continue
					}

					prefix := pod.Name
					if componentLabel, ok := pod.Labels[k8sComponentLabel]; ok {
						prefix = componentLabel
					}

					if printInfo {
						log.Info("Starting log streaming for containers that use images defined in devspace.yaml\n")
						printInfo = false
					}

					wg.Add(1)
					go func(prefix string, reader io.Reader, color string) {
						scanner := bufio.NewScanner(reader)
						for scanner.Scan() {
							lines <- &logLine{
								line:  scanner.Text(),
								name:  prefix,
								color: color,
							}
						}

						wg.Done()
					}(prefix, reader, logpkg.Colors[idx%len(logpkg.Colors)])
					break Outer
				}
			}
		}
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	for {
		select {
		case err := <-interrupt:
			cancelFunc()
			return err
		case <-done:
			return nil
		case line := <-lines:
			writer.Write([]byte(ansi.Color(fmt.Sprintf("[%s]", line.name), line.color) + " " + line.line + "\n"))
		}
	}
}

// LogMultiple will log multiple
func (client *client) LogMultiple(imageSelector []string, interrupt chan error, tail *int64, writer io.Writer, log logpkg.Logger) error {
	return client.LogMultipleTimeout(imageSelector, interrupt, tail, writer, time.Minute*2, log)
}

// Logs prints the container logs
func (client *client) Logs(ctx context.Context, namespace, podName, containerName string, lastContainerLog bool, tail *int64, follow bool) (io.ReadCloser, error) {
	lines := int64(100)
	if tail != nil {
		lines = *tail
	}

	request := client.KubeClient().CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{
		Container: containerName,
		TailLines: &lines,
		Previous:  lastContainerLog,
		Follow:    follow,
	})

	if request.URL().String() == "" {
		return nil, errors.New("Request url is empty")
	}

	return request.Stream(ctx)
}
