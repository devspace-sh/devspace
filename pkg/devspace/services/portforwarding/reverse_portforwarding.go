package portforwarding

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/services/sync"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"io"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/services/inject"
	"github.com/loft-sh/devspace/pkg/devspace/tunnel"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/pkg/errors"
)

func startReversePortForwarding(ctx *devspacecontext.Context, name, arch string, portForwarding []*latest.PortMapping, selector targetselector.TargetSelector, parent *tomb.Tomb) error {
	if ctx.IsDone() {
		return nil
	}

	fileLog := logpkg.GetDevPodFileLogger(name)
	container, err := selector.SelectSingleContainer(ctx.Context, ctx.KubeClient, ctx.Log)
	if err != nil {
		return errors.Wrap(err, "error selecting container")
	}

	// make sure the DevSpace helper binary is injected
	ctx.Log.Info("Reverse-Port-Forwarding: Inject devspacehelper...")
	err = inject.InjectDevSpaceHelper(ctx.Context, ctx.KubeClient, container.Pod, container.Container.Name, arch, ctx.Log)
	if err != nil {
		return err
	}

	errorChan := make(chan error, 2)
	closeChan := make(chan struct{})

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	go func() {
		err := sync.StartStream(ctx.Context, ctx.KubeClient, container.Pod, container.Container.Name, []string{inject.DevSpaceHelperContainerPath, "tunnel"}, stdinReader, stdoutWriter, false, fileLog)
		if err != nil {
			errorChan <- errors.Errorf("connection lost to pod %s/%s: %v", container.Pod.Namespace, container.Pod.Name, err)
		}
	}()

	go func() {
		err := tunnel.StartReverseForward(ctx.Context, stdoutReader, stdinWriter, portForwarding, closeChan, container.Pod.Namespace, container.Pod.Name, ctx.Log)
		if err != nil {
			errorChan <- err
		}
	}()

	parent.Go(func() error {
		select {
		case <-ctx.Context.Done():
			close(closeChan)
			_ = stdinWriter.Close()
			_ = stdoutWriter.Close()
			doneReverseForwarding(ctx, name, portForwarding, fileLog, parent)
		case err := <-errorChan:
			if err != nil {
				fileLog.Errorf("Reverse portforwarding restarting, because: %v", err)
				sync.PrintPodError(ctx.Context, ctx.KubeClient, container.Pod, fileLog)
				close(closeChan)
				_ = stdinWriter.Close()
				_ = stdoutWriter.Close()
				hook.LogExecuteHooks(ctx.WithLogger(fileLog), map[string]interface{}{
					"reverse_port_forwarding_config": portForwarding,
					"error":                          err,
				}, hook.EventsForSingle("restart:reversePortForwarding", name).With("reversePortForwarding.restart")...)

				for {
					err = startReversePortForwarding(ctx.WithLogger(fileLog), name, arch, portForwarding, selector, parent)
					if err != nil {
						hook.LogExecuteHooks(ctx.WithLogger(fileLog), map[string]interface{}{
							"reverse_port_forwarding_config": portForwarding,
							"error":                          err,
						}, hook.EventsForSingle("restart:reversePortForwarding", name).With("reversePortForwarding.restart")...)
						fileLog.Errorf("Error restarting reverse port-forwarding: %v", err)
						fileLog.Errorf("Will try again in 15 seconds")

						select {
						case <-time.After(time.Second * 15):
							continue
						case <-ctx.Context.Done():
							doneReverseForwarding(ctx, name, portForwarding, fileLog, parent)
							return nil
						}
					}

					break
				}
			}
		}
		return nil
	})

	return nil
}

func doneReverseForwarding(ctx *devspacecontext.Context, name string, portForwarding []*latest.PortMapping, fileLog logpkg.Logger, parent *tomb.Tomb) {
	hook.LogExecuteHooks(ctx.WithLogger(fileLog), map[string]interface{}{
		"reverse_port_forwarding_config": portForwarding,
	}, hook.EventsForSingle("stop:reversePortForwarding", name).With("reversePortForwarding.stop")...)
	fileLog.Done("Stopped reverse port forwarding %s", name)
	parent.Kill(nil)
}
