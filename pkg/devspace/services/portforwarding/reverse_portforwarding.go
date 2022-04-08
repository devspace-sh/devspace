package portforwarding

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/services/sync"
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

func StartReversePortForwarding(ctx devspacecontext.Context, name, arch string, portForwarding []*latest.PortMapping, selector targetselector.TargetSelector, parent *tomb.Tomb) error {
	if ctx.IsDone() {
		return nil
	}

	container, err := selector.SelectSingleContainer(ctx.Context(), ctx.KubeClient(), ctx.Log())
	if err != nil {
		return errors.Wrap(err, "error selecting container")
	}

	// make sure the DevSpace helper binary is injected
	err = inject.InjectDevSpaceHelper(ctx.Context(), ctx.KubeClient(), container.Pod, container.Container.Name, arch, ctx.Log())
	if err != nil {
		return err
	}

	errorChan := make(chan error, 2)
	closeChan := make(chan struct{})

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	go func() {
		err := sync.StartStream(ctx.Context(), ctx.KubeClient(), container.Pod, container.Container.Name, []string{inject.DevSpaceHelperContainerPath, "tunnel"}, stdinReader, stdoutWriter, false, ctx.Log())
		if err != nil {
			errorChan <- errors.Errorf("connection lost to pod %s/%s: %v", container.Pod.Namespace, container.Pod.Name, err)
		}
	}()

	go func() {
		err := tunnel.StartReverseForward(ctx.Context(), stdoutReader, stdinWriter, portForwarding, closeChan, container.Pod.Namespace, container.Pod.Name, ctx.Log())
		if err != nil {
			errorChan <- err
		}
	}()

	parent.Go(func() error {
		select {
		case <-ctx.Context().Done():
			close(closeChan)
			_ = stdinWriter.Close()
			_ = stdoutWriter.Close()
			doneReverseForwarding(ctx, name, portForwarding, parent)
		case err := <-errorChan:
			if ctx.IsDone() {
				close(closeChan)
				_ = stdinWriter.Close()
				_ = stdoutWriter.Close()
				doneReverseForwarding(ctx, name, portForwarding, parent)
				return nil
			}
			if err != nil {
				ctx.Log().Errorf("Restarting because: %v", err)
				shouldExit := sync.PrintPodError(ctx.Context(), ctx.KubeClient(), container.Pod, ctx.Log())
				close(closeChan)
				_ = stdinWriter.Close()
				_ = stdoutWriter.Close()
				hook.LogExecuteHooks(ctx, map[string]interface{}{
					"reverse_port_forwarding_config": portForwarding,
					"error":                          err,
				}, hook.EventsForSingle("restart:reversePortForwarding", name).With("reversePortForwarding.restart")...)
				if shouldExit {
					doneReverseForwarding(ctx, name, portForwarding, parent)
					return nil
				}

				for {
					err = StartReversePortForwarding(ctx, name, arch, portForwarding, selector, parent)
					if err != nil {
						hook.LogExecuteHooks(ctx, map[string]interface{}{
							"reverse_port_forwarding_config": portForwarding,
							"error":                          err,
						}, hook.EventsForSingle("restart:reversePortForwarding", name).With("reversePortForwarding.restart")...)
						ctx.Log().Errorf("Error restarting reverse port-forwarding: %v", err)
						ctx.Log().Errorf("Will try again in 15 seconds")

						select {
						case <-time.After(time.Second * 15):
							continue
						case <-ctx.Context().Done():
							doneReverseForwarding(ctx, name, portForwarding, parent)
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

func doneReverseForwarding(ctx devspacecontext.Context, name string, portForwarding []*latest.PortMapping, parent *tomb.Tomb) {
	hook.LogExecuteHooks(ctx, map[string]interface{}{
		"reverse_port_forwarding_config": portForwarding,
	}, hook.EventsForSingle("stop:reversePortForwarding", name).With("reversePortForwarding.stop")...)
	parent.Kill(nil)
	for _, m := range portForwarding {
		ctx.Log().Debugf("Stopped reverse port forwarding %v", m.Port)
	}
}
