package services

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/util"
	"github.com/loft-sh/devspace/pkg/devspace/services/inject"
	"github.com/loft-sh/devspace/pkg/devspace/tunnel"
	"github.com/loft-sh/devspace/pkg/util/imageselector"
	"io"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
)

// StartReversePortForwarding starts the reverse port forwarding functionality
func (serviceClient *client) StartReversePortForwarding(interrupt chan error) error {
	if serviceClient.config == nil || serviceClient.config.Config() == nil || serviceClient.config.Generated() == nil {
		return fmt.Errorf("DevSpace config is not set")
	}

	cache := serviceClient.config.Generated().GetActive()
	for _, portForwarding := range serviceClient.config.Config().Dev.Ports {
		if len(portForwarding.PortMappingsReverse) == 0 {
			continue
		}

		// start reverse port forwarding
		err := serviceClient.startReversePortForwarding(cache, portForwarding, interrupt, serviceClient.log)
		if err != nil {
			return err
		}
	}

	return nil
}

func (serviceClient *client) startReversePortForwarding(cache *generated.CacheConfig, portForwarding *latest.PortForwardingConfig, interrupt chan error, log logpkg.Logger) error {
	var err error

	// apply config & set image selector
	options := targetselector.NewEmptyOptions().ApplyConfigParameter(portForwarding.LabelSelector, portForwarding.Namespace, portForwarding.ContainerName, "")
	options.AllowPick = false
	options.ImageSelector = []imageselector.ImageSelector{}
	imageSelector, err := imageselector.Resolve(portForwarding.ImageName, serviceClient.config, serviceClient.dependencies)
	if err != nil {
		return err
	} else if imageSelector != nil {
		options.ImageSelector = append(options.ImageSelector, *imageSelector)
	}
	if portForwarding.ImageSelector != "" {
		imageSelector, err := util.ResolveImageAsImageSelector(portForwarding.ImageSelector, serviceClient.config, serviceClient.dependencies)
		if err != nil {
			return err
		}

		options.ImageSelector = append(options.ImageSelector, *imageSelector)
	}
	options.WaitingStrategy = targetselector.NewUntilNewestRunningWaitingStrategy(time.Second * 2)
	options.SkipInitContainers = true

	log.StartWait("Reverse-Port-Forwarding: Waiting for containers to start...")
	container, err := targetselector.NewTargetSelector(serviceClient.client).SelectSingleContainer(context.TODO(), options, log)
	log.StopWait()
	if err != nil {
		return errors.Errorf("%s: %s", message.SelectorErrorPod, err.Error())
	}

	// make sure the devspace helper binary is injected
	log.StartWait("Reverse-Port-Forwarding: Upload devspace helper...")
	err = inject.InjectDevSpaceHelper(serviceClient.client, container.Pod, container.Container.Name, string(portForwarding.Arch), serviceClient.log)
	log.StopWait()
	if err != nil {
		return err
	}

	errorChan := make(chan error, 2)
	closeChan := make(chan error)

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	go func() {
		err := inject.StartStream(serviceClient.client, container.Pod, container.Container.Name, []string{inject.DevSpaceHelperContainerPath, "tunnel"}, stdinReader, stdoutWriter)
		if err != nil {
			errorChan <- errors.Errorf("Reverse Port Forwarding - connection lost to pod %s/%s: %v", container.Pod.Namespace, container.Pod.Name, err)
		}
	}()

	go func() {
		err := tunnel.StartReverseForward(stdoutReader, stdinWriter, portForwarding.PortMappingsReverse, closeChan, container.Pod.Namespace, container.Pod.Name, log)
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
					err = serviceClient.startReversePortForwarding(cache, portForwarding, interrupt, logpkg.Discard)
					if err != nil {
						serviceClient.log.Errorf("Error restarting reverse port-forwarding: %v", err)
						serviceClient.log.Errorf("Will try again in 15 seconds")
						time.Sleep(time.Second * 15)
						continue
					}

					time.Sleep(time.Second * 5)
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
