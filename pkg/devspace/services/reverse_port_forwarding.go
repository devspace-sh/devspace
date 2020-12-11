package services

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/tunnel"
	"io"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/pkg/errors"
)

// StartPortForwarding starts the port forwarding functionality
func (serviceClient *client) StartReversePortForwarding(interrupt chan error) error {
	if serviceClient.config.Dev == nil {
		return nil
	}

	for _, portForwarding := range serviceClient.config.Dev.Ports {
		if len(portForwarding.PortMappingsReverse) == 0 {
			continue
		}

		err := serviceClient.startReversePortForwarding(portForwarding, interrupt, serviceClient.log)
		if err != nil {
			return err
		}
	}

	return nil
}

func (serviceClient *client) startReversePortForwarding(portForwarding *latest.PortForwardingConfig, interrupt chan error, log logpkg.Logger) error {
	selector, err := targetselector.NewTargetSelector(serviceClient.client, &targetselector.SelectorParameter{
		ConfigParameter: targetselector.ConfigParameter{
			Namespace:     portForwarding.Namespace,
			LabelSelector: portForwarding.LabelSelector,
			ContainerName: portForwarding.ContainerName,
		},
	}, false, targetselector.ImageSelectorFromConfig(portForwarding.ImageName, serviceClient.config, serviceClient.generated))
	if err != nil {
		return errors.Errorf("Error creating target selector: %v", err)
	}

	log.StartWait("Reverse-Port-Forwarding: Waiting for containers to start...")
	pod, container, err := selector.GetContainer(false, log)
	log.StopWait()
	if err != nil {
		return errors.Errorf("%s: %s", message.SelectorErrorPod, err.Error())
	}

	// make sure the devspace helper binary is injected
	err = InjectDevSpaceHelper(serviceClient.client, pod, container.Name, serviceClient.log)
	if err != nil {
		return err
	}

	errorChan := make(chan error, 2)
	closeChan := make(chan error)

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	go func() {
		err := serviceClient.startStream(pod, container.Name, []string{DevSpaceHelperContainerPath, "tunnel"}, stdinReader, stdoutWriter)
		if err != nil {
			errorChan <- errors.Errorf("Reverse Port Forwarding - connection lost to pod %s/%s: %v", pod.Namespace, pod.Name, err)
		}
	}()

	go func() {
		err := tunnel.StartReverseForward(stdoutReader, stdinWriter, portForwarding.PortMappingsReverse, closeChan, log)
		if err != nil {
			errorChan <- err
		}
	}()

	logFile := logpkg.GetFileLogger("reverse-portforwarding")
	go func(portForwarding *latest.PortForwardingConfig, interrupt chan error) {
		select {
		case err := <-errorChan:
			if err != nil {
				close(closeChan)
				stdinWriter.Close()
				stdoutWriter.Close()
				logFile.Error(err)
				for {
					err = serviceClient.startReversePortForwarding(portForwarding, interrupt, logpkg.Discard)
					if err != nil {
						serviceClient.log.Errorf("Error restarting reverse port-forwarding: %v", err)
						serviceClient.log.Errorf("Will try again in 3 seconds")
						continue
					}

					time.Sleep(time.Second * 3)
					break
				}
			}
		case <-interrupt:
			close(closeChan)
			stdinWriter.Close()
			stdoutWriter.Close()
		}
	}(portForwarding, interrupt)

	return nil
}
