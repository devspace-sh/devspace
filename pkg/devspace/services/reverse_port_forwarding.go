package services

import (
	"context"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
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

	var cache *generated.CacheConfig
	if serviceClient.generated != nil {
		cache = serviceClient.generated.GetActive()
	}

	options := targetselector.NewEmptyOptions()
	options.AllowPick = false
	for _, portForwarding := range serviceClient.config.Dev.Ports {
		if len(portForwarding.PortMappingsReverse) == 0 {
			continue
		}

		// apply config & set image selector
		newOptions := options.ApplyConfigParameter(portForwarding.LabelSelector, portForwarding.Namespace, portForwarding.ContainerName, "")
		newOptions.ImageSelector = targetselector.ImageSelectorFromConfig(portForwarding.ImageName, serviceClient.config, cache)
		newOptions.WaitingStrategy = targetselector.NewUntilNewestRunningWaitingStrategy(time.Second * 2)
		newOptions.SkipInitContainers = true

		// start reverse portforwarding
		err := serviceClient.startReversePortForwarding(newOptions, portForwarding, interrupt, serviceClient.log)
		if err != nil {
			return err
		}
	}

	return nil
}

func (serviceClient *client) startReversePortForwarding(options targetselector.Options, portForwarding *latest.PortForwardingConfig, interrupt chan error, log logpkg.Logger) error {
	log.StartWait("Reverse-Port-Forwarding: Waiting for containers to start...")
	container, err := targetselector.NewTargetSelector(serviceClient.client).SelectSingleContainer(context.TODO(), options, log)
	log.StopWait()
	if err != nil {
		return errors.Errorf("%s: %s", message.SelectorErrorPod, err.Error())
	}

	// make sure the devspace helper binary is injected
	err = InjectDevSpaceHelper(serviceClient.client, container.Pod, container.Container.Name, serviceClient.log)
	if err != nil {
		return err
	}

	errorChan := make(chan error, 2)
	closeChan := make(chan error)

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	go func() {
		err := serviceClient.startStream(container.Pod, container.Container.Name, []string{DevSpaceHelperContainerPath, "tunnel"}, stdinReader, stdoutWriter)
		if err != nil {
			errorChan <- errors.Errorf("Reverse Port Forwarding - connection lost to pod %s/%s: %v", container.Pod.Namespace, container.Pod.Name, err)
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
					err = serviceClient.startReversePortForwarding(options, portForwarding, interrupt, logpkg.Discard)
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
