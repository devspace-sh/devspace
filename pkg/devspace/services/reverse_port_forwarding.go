package services

import (
	"context"
	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"io"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/services/inject"
	"github.com/loft-sh/devspace/pkg/devspace/services/synccontroller"
	"github.com/loft-sh/devspace/pkg/devspace/tunnel"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
)

func (serviceClient *client) startReversePortForwarding(portForwarding *latest.PortForwardingConfig, interrupt chan error, fileLog, log logpkg.Logger) error {
	var (
		err           error
		imageSelector []string
	)
	if portForwarding.ImageSelector != "" {
		imageSelectorObject, err := runtimevar.NewRuntimeResolver(true).FillRuntimeVariablesAsImageSelector(portForwarding.ImageSelector, serviceClient.config, serviceClient.dependencies)
		if err != nil {
			return err
		}

		imageSelector = []string{imageSelectorObject.Image}
	}

	// apply config & set image selector
	options := targetselector.NewEmptyOptions().
		ApplyConfigParameter(portForwarding.ContainerName, portForwarding.LabelSelector, imageSelector, portForwarding.Namespace, "").
		WithWaitingStrategy(targetselector.NewUntilNewestRunningWaitingStrategy(time.Second * 2)).
		WithSkipInitContainers(true)

	log.Info("Reverse-Port-Forwarding: Waiting for containers to start...")
	container, err := targetselector.GlobalTargetSelector.SelectSingleContainer(context.TODO(), serviceClient.client, options, log)
	if err != nil {
		return errors.Errorf("%s: %s", message.SelectorErrorPod, err.Error())
	}

	// make sure the devspace helper binary is injected
	log.Info("Reverse-Port-Forwarding: Inject devspacehelper...")
	err = inject.InjectDevSpaceHelper(serviceClient.client, container.Pod, container.Container.Name, string(portForwarding.Arch), log)
	if err != nil {
		return err
	}

	errorChan := make(chan error, 2)
	closeChan := make(chan error)

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	go func() {
		err := synccontroller.StartStream(serviceClient.client, container.Pod, container.Container.Name, []string{inject.DevSpaceHelperContainerPath, "tunnel"}, stdinReader, stdoutWriter, false, fileLog)
		if err != nil {
			errorChan <- errors.Errorf("connection lost to pod %s/%s: %v", container.Pod.Namespace, container.Pod.Name, err)
		}
	}()

	go func() {
		err := tunnel.StartReverseForward(stdoutReader, stdinWriter, portForwarding.PortMappingsReverse, closeChan, container.Pod.Namespace, container.Pod.Name, log)
		if err != nil {
			errorChan <- err
		}
	}()

	go func(portForwarding *latest.PortForwardingConfig, interrupt chan error) {
		select {
		case err := <-errorChan:
			if err != nil {
				fileLog.Errorf("Reverse portforwarding restarting, because: %v", err)
				synccontroller.PrintPodError(context.TODO(), serviceClient.KubeClient(), container.Pod, fileLog)
				close(closeChan)
				_ = stdinWriter.Close()
				_ = stdoutWriter.Close()
				hook.LogExecuteHooks(serviceClient.KubeClient(), serviceClient.Config(), serviceClient.Dependencies(), map[string]interface{}{
					"reverse_port_forwarding_config": portForwarding,
					"error":                          err,
				}, fileLog, hook.EventsForSingle("restart:reversePortForwarding", portForwarding.Name).With("reversePortForwarding.restart")...)

				for {
					err = serviceClient.startReversePortForwarding(portForwarding, interrupt, fileLog, fileLog)
					if err != nil {
						hook.LogExecuteHooks(serviceClient.KubeClient(), serviceClient.Config(), serviceClient.Dependencies(), map[string]interface{}{
							"reverse_port_forwarding_config": portForwarding,
							"error":                          err,
						}, fileLog, hook.EventsForSingle("restart:reversePortForwarding", portForwarding.Name).With("reversePortForwarding.restart")...)
						fileLog.Errorf("Error restarting reverse port-forwarding: %v", err)
						fileLog.Errorf("Will try again in 15 seconds")
						time.Sleep(time.Second * 15)
						continue
					}

					time.Sleep(time.Second * 5)
					break
				}
			}
		case <-interrupt:
			close(closeChan)
			_ = stdinWriter.Close()
			_ = stdoutWriter.Close()
			hook.LogExecuteHooks(serviceClient.KubeClient(), serviceClient.Config(), serviceClient.Dependencies(), map[string]interface{}{
				"reverse_port_forwarding_config": portForwarding,
			}, fileLog, hook.EventsForSingle("stop:reversePortForwarding", portForwarding.Name).With("reversePortForwarding.stop")...)
			fileLog.Done("Stopped reverse port forwarding %s", portForwarding.Name)
		}
	}(portForwarding, interrupt)

	return nil
}
