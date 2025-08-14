package portforwarding

import (
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/devspace/helper/util/port"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/portforward"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/mgutz/ansi"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/services/sync"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/pkg/errors"
)

// StartPortForwarding starts the port forwarding functionality
func StartPortForwarding(ctx devspacecontext.Context, devPod *latest.DevPod, selector targetselector.TargetSelector, parent *tomb.Tomb) (retErr error) {
	if ctx == nil || ctx.Config() == nil || ctx.Config().Config() == nil {
		return fmt.Errorf("DevSpace config is not set")
	}

	// forward
	initDoneArray := []chan struct{}{}
	if len(devPod.Ports) > 0 {
		initDoneArray = append(initDoneArray, parent.NotifyGo(func() error {
			return startPortForwardingWithHooks(ctx, devPod.Name, devPod.Ports, selector, parent)
		}))
	}

	// reverse
	loader.EachDevContainer(devPod, func(devContainer *latest.DevContainer) bool {
		if len(devContainer.ReversePorts) > 0 {
			initDoneArray = append(initDoneArray, parent.NotifyGo(func() error {
				return startReversePortForwardingWithHooks(ctx, devPod.Name, string(devContainer.Arch), devContainer.ReversePorts, selector.WithContainer(devContainer.Container), parent)
			}))
		}
		return true
	})

	// wait until everything is initialized
	for _, initDone := range initDoneArray {
		<-initDone
	}
	return nil
}

func startReversePortForwardingWithHooks(ctx devspacecontext.Context, name, arch string, portMappings []*latest.PortMapping, selector targetselector.TargetSelector, parent *tomb.Tomb) error {
	pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
		"reverse_port_forwarding_config": portMappings,
	}, hook.EventsForSingle("start:reversePortForwarding", name).With("reversePortForwarding.start")...)
	if pluginErr != nil {
		return pluginErr
	}

	// start reverse port forwarding
	err := StartReversePortForwarding(ctx, name, arch, portMappings, selector, parent)
	if err != nil {
		pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
			"reverse_port_forwarding_config": portMappings,
			"error":                          err,
		}, hook.EventsForSingle("error:reversePortForwarding", name).With("reversePortForwarding.error")...)
		if pluginErr != nil {
			return pluginErr
		}

		return err
	}

	return nil
}

func startPortForwardingWithHooks(ctx devspacecontext.Context, name string, portMappings []*latest.PortMapping, selector targetselector.TargetSelector, parent *tomb.Tomb) error {
	pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
		"port_forwarding_config": portMappings,
	}, hook.EventsForSingle("start:portForwarding", name).With("portForwarding.start")...)
	if pluginErr != nil {
		return pluginErr
	}

	// start port forwarding
	err := StartForwarding(ctx, name, portMappings, selector, parent)
	if err != nil {
		pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
			"port_forwarding_config": portMappings,
			"error":                  err,
		}, hook.EventsForSingle("error:portForwarding", name).With("portForwarding.error")...)
		if pluginErr != nil {
			return pluginErr
		}

		return err
	}

	return nil
}

func StartForwarding(ctx devspacecontext.Context, name string, portMappings []*latest.PortMapping, selector targetselector.TargetSelector, parent *tomb.Tomb) error {
	if ctx.IsDone() {
		return nil
	}

	// start port forwarding
	pod, err := selector.SelectSinglePod(ctx.Context(), ctx.KubeClient(), ctx.Log())
	if err != nil {
		return errors.Wrap(err, "error selecting pod")
	} else if pod == nil {
		return nil
	}

	ports := make([]string, len(portMappings))
	portsFormatted := make([]string, len(portMappings))
	addresses := make([]string, len(portMappings))
	for index, value := range portMappings {
		if value.Port == "" {
			return errors.Errorf("port is not defined in portmapping %d", index)
		}

		mappings, err := portforward.ParsePorts([]string{value.Port})
		if err != nil {
			return fmt.Errorf("error parsing port %s: %v", value.Port, err)
		}

		localPort := mappings[0].Local
		remotePort := mappings[0].Remote
		available, err := port.IsAvailable(fmt.Sprintf(":%d", int(localPort)))
		if err != nil {
			ctx.Log().Debugf("Seems like port %d is already in use: %v", err)
		} else if !available {
			ctx.Log().Debugf("Seems like port %d is already in use. Is another application using that port?", localPort)
		}

		ports[index] = fmt.Sprintf("%d:%d", int(localPort), int(remotePort))
		portsFormatted[index] = ansi.Color(fmt.Sprintf("%d -> %d", int(localPort), int(remotePort)), "white+b")
		if value.BindAddress == "" {
			addresses[index] = "localhost"
		} else {
			addresses[index] = value.BindAddress
		}
	}

	readyChan := make(chan struct{})
	errorChan := make(chan error, 1)
	pf, err := kubectl.NewPortForwarder(ctx.KubeClient(), pod, ports, addresses, make(chan struct{}), readyChan, errorChan)
	if err != nil {
		return errors.Errorf("Error starting port forwarding: %v", err)
	}

	go func() {
		err := pf.ForwardPorts(ctx.Context())
		if err != nil {
			errorChan <- err
		}
	}()

	// Wait till forwarding is ready
	select {
	case <-ctx.Context().Done():
		return nil
	case <-readyChan:
		ctx.Log().Donef("Port forwarding started on: %s", strings.Join(portsFormatted, ", "))
	case err := <-errorChan:
		if ctx.IsDone() {
			return nil
		}

		return errors.Wrap(err, "forward ports")
	case <-time.After(20 * time.Second):
		return errors.Errorf("Timeout waiting for port forwarding to start")
	}

	parent.Go(func() error {
		select {
		case <-ctx.Context().Done():
			pf.Close()
			stopPortForwarding(ctx, name, portMappings, parent)
		case err := <-errorChan:
			if ctx.IsDone() {
				pf.Close()
				stopPortForwarding(ctx, name, portMappings, parent)
				return nil
			}
			if err != nil {
				ctx.Log().Errorf("Restarting because: %v", err)
				shouldExit := sync.PrintPodError(ctx.Context(), ctx.KubeClient(), pod, ctx.Log())
				pf.Close()
				hook.LogExecuteHooks(ctx, map[string]interface{}{
					"port_forwarding_config": portMappings,
					"error":                  err,
				}, hook.EventsForSingle("restart:portForwarding", name).With("portForwarding.restart")...)
				if shouldExit {
					stopPortForwarding(ctx, name, portMappings, parent)
					return nil
				}

				for {
					err = StartForwarding(ctx, name, portMappings, selector, parent)
					if err != nil {
						hook.LogExecuteHooks(ctx, map[string]interface{}{
							"port_forwarding_config": portMappings,
							"error":                  err,
						}, hook.EventsForSingle("restart:portForwarding", name).With("portForwarding.restart")...)
						ctx.Log().Errorf("Error restarting port-forwarding: %v", err)
						ctx.Log().Errorf("Will try again in 15 seconds")

						select {
						case <-time.After(time.Second * 15):
							continue
						case <-ctx.Context().Done():
							stopPortForwarding(ctx, name, portMappings, parent)
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

func stopPortForwarding(ctx devspacecontext.Context, name string, portMappings []*latest.PortMapping, parent *tomb.Tomb) {
	hook.LogExecuteHooks(ctx, map[string]interface{}{
		"port_forwarding_config": portMappings,
	}, hook.EventsForSingle("stop:portForwarding", name).With("portForwarding.stop")...)
	parent.Kill(nil)
	for _, m := range portMappings {
		ctx.Log().Debugf("Stopped port forwarding %v", m.Port)
	}
}
