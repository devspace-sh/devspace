package portforwarding

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	runner2 "github.com/loft-sh/devspace/pkg/devspace/services/runner"
	"github.com/loft-sh/devspace/pkg/devspace/services/sync"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/loft-sh/devspace/pkg/util/port"
	"github.com/pkg/errors"
)

// StartPortForwarding starts the port forwarding functionality
func StartPortForwarding(ctx *devspacecontext.Context, devPod *latest.DevPod, selector targetselector.TargetSelector, done chan struct{}) error {
	if ctx == nil || ctx.Config == nil || ctx.Config.Config() == nil {
		return fmt.Errorf("DevSpace config is not set")
	}

	runner := runner2.NewRunner(5)

	// forward
	doneChans := []chan struct{}{}
	if len(devPod.Forward) > 0 {
		doneChan := make(chan struct{})
		doneChans = append(doneChans, doneChan)
		err := runner.Run(newPortForwardingFn(ctx, devPod.Name, devPod.Forward, selector, doneChan))
		if err != nil {
			if done != nil {
				close(done)
			}
			return err
		}
	}

	// reverse
	var err error
	loader.EachDevContainer(devPod, func(devContainer *latest.DevContainer) bool {
		if len(devPod.PortMappingsReverse) > 0 {
			doneChan := make(chan struct{})
			doneChans = append(doneChans, doneChan)
			err = runner.Run(newReversePortForwardingFn(ctx, devPod.Name, string(devContainer.Arch), devContainer.PortMappingsReverse, selector.WithContainer(devContainer.Container), doneChan))
			if err != nil {
				if done != nil {
					close(done)
				}
				return false
			}
		}
		return true
	})
	if err != nil {
		return err
	}

	if done != nil {
		go func() {
			for i := 0; i < len(doneChans); i++ {
				<-doneChans[i]
			}

			select {
			case <-done:
			default:
				close(done)
			}
		}()
	}

	return runner.Wait()
}

func newReversePortForwardingFn(ctx *devspacecontext.Context, name, arch string, portMappings []*latest.PortMapping, selector targetselector.TargetSelector, done chan struct{}) func() error {
	return func() error {
		pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
			"reverse_port_forwarding_config": portMappings,
		}, hook.EventsForSingle("start:reversePortForwarding", name).With("reversePortForwarding.start")...)
		if pluginErr != nil {
			return pluginErr
		}

		// start reverse port forwarding
		err := startReversePortForwarding(ctx, name, arch, portMappings, selector, done)
		if err != nil {
			pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
				"reverse_port_forwarding_config": portMappings,
				"error":                          err,
			}, hook.EventsForSingle("error:reversePortForwarding", name).With("reversePortForwarding.error")...)
			if pluginErr != nil {
				return pluginErr
			}

			close(done)
			return err
		}

		return nil
	}
}

func newPortForwardingFn(ctx *devspacecontext.Context, name string, portMappings []*latest.PortMapping, selector targetselector.TargetSelector, done chan struct{}) func() error {
	return func() error {
		pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
			"port_forwarding_config": portMappings,
		}, hook.EventsForSingle("start:portForwarding", name).With("portForwarding.start")...)
		if pluginErr != nil {
			return pluginErr
		}

		// start port forwarding
		err := startForwarding(ctx, name, portMappings, selector, done)
		if err != nil {
			pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
				"port_forwarding_config": portMappings,
				"error":                  err,
			}, hook.EventsForSingle("error:portForwarding", name).With("portForwarding.error")...)
			if pluginErr != nil {
				return pluginErr
			}

			close(done)
			return err
		}

		return nil
	}
}

func startForwarding(ctx *devspacecontext.Context, name string, portMappings []*latest.PortMapping, selector targetselector.TargetSelector, done chan struct{}) error {
	// start port forwarding
	pod, err := selector.SelectSinglePod(ctx.Context, ctx.KubeClient, ctx.Log)
	if err != nil {
		return errors.Errorf("%s: %s", message.SelectorErrorPod, err.Error())
	} else if pod == nil {
		return nil
	}

	ports := make([]string, len(portMappings))
	addresses := make([]string, len(portMappings))
	for index, value := range portMappings {
		if value.LocalPort == nil {
			return errors.Errorf("port is not defined in portmapping %d", index)
		}

		localPort := strconv.Itoa(*value.LocalPort)
		remotePort := localPort
		if value.RemotePort != nil {
			remotePort = strconv.Itoa(*value.RemotePort)
		}

		open, _ := port.Check(*value.LocalPort)
		if !open {
			ctx.Log.Warnf("Seems like port %d is already in use. Is another application using that port?", *value.LocalPort)
		}

		ports[index] = localPort + ":" + remotePort
		if value.BindAddress == "" {
			addresses[index] = "localhost"
		} else {
			addresses[index] = value.BindAddress
		}
	}

	readyChan := make(chan struct{})
	errorChan := make(chan error)
	pf, err := ctx.KubeClient.NewPortForwarder(pod, ports, addresses, make(chan struct{}), readyChan, errorChan)
	if err != nil {
		return errors.Errorf("Error starting port forwarding: %v", err)
	}

	go func() {
		err := pf.ForwardPorts(ctx.Context)
		if err != nil {
			errorChan <- err
		}
	}()

	// Wait till forwarding is ready
	select {
	case <-ctx.Context.Done():
		return nil
	case <-readyChan:
		ctx.Log.Donef("Port forwarding started on %s (%s/%s)", strings.Join(ports, ", "), pod.Namespace, pod.Name)
	case err := <-errorChan:
		return errors.Wrap(err, "forward ports")
	case <-time.After(20 * time.Second):
		return errors.Errorf("Timeout waiting for port forwarding to start")
	}

	go func(portMappings []*latest.PortMapping) {
		fileLog := logpkg.GetDevPodFileLogger(name)
		select {
		case <-ctx.Context.Done():
			pf.Close()
			hook.LogExecuteHooks(ctx.WithLogger(fileLog), map[string]interface{}{
				"port_forwarding_config": portMappings,
			}, hook.EventsForSingle("stop:portForwarding", name).With("portForwarding.stop")...)
			close(done)
			fileLog.Done("Stopped port forwarding")
		case err := <-errorChan:
			if err != nil {
				fileLog.Errorf("Portforwarding restarting, because: %v", err)
				sync.PrintPodError(context.TODO(), ctx.KubeClient, pod, fileLog)
				pf.Close()
				hook.LogExecuteHooks(ctx.WithLogger(fileLog), map[string]interface{}{
					"port_forwarding_config": portMappings,
					"error":                  err,
				}, hook.EventsForSingle("restart:portForwarding", name).With("portForwarding.restart")...)

				for {
					err = startForwarding(ctx.WithLogger(fileLog), name, portMappings, selector, done)
					if err != nil {
						hook.LogExecuteHooks(ctx.WithLogger(fileLog), map[string]interface{}{
							"port_forwarding_config": portMappings,
							"error":                  err,
						}, hook.EventsForSingle("restart:portForwarding", name).With("portForwarding.restart")...)
						fileLog.Errorf("Error restarting port-forwarding: %v", err)
						fileLog.Errorf("Will try again in 15 seconds")
						time.Sleep(time.Second * 15)
						continue
					}

					time.Sleep(time.Second * 3)
					break
				}
			}
		}
	}(portMappings)

	return nil
}
