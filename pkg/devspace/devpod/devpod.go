package devpod

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/services/logs"
	"github.com/loft-sh/devspace/pkg/devspace/services/terminal"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"os"
	syncpkg "sync"

	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/services/podreplace"
	"github.com/loft-sh/devspace/pkg/devspace/services/portforwarding"
	"github.com/loft-sh/devspace/pkg/devspace/services/sync"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/pkg/errors"
	"time"
)

type devPod struct {
	selectedPod *corev1.Pod

	m syncpkg.Mutex

	done chan struct{}

	cancelCtx context.Context
	cancel    context.CancelFunc
}

func newDevPod() *devPod {
	return &devPod{
		done: make(chan struct{}),
	}
}

func (d *devPod) Start(ctx *devspacecontext.Context, devPodConfig *latest.DevPod, options Options) error {
	d.m.Lock()
	defer d.m.Unlock()

	if d.cancel != nil {
		return errors.Errorf("dev pod is already running, please stop it before starting")
	}
	d.cancelCtx, d.cancel = context.WithCancel(ctx.Context)
	ctx = ctx.WithContext(d.cancelCtx)

	// start the dev pod
	err := d.startWithRetry(ctx, devPodConfig, options)
	if err != nil {
		d.cancel()
		<-d.done
		return err
	}

	return nil
}

func (d *devPod) Done() <-chan struct{} {
	return d.done
}

func (d *devPod) Stop() {
	d.m.Lock()
	defer d.m.Lock()

	if d.cancel != nil {
		d.cancel()
		<-d.done
	}
}

func (d *devPod) startWithRetry(ctx *devspacecontext.Context, devPodConfig *latest.DevPod, options Options) error {
	t := &tomb.Tomb{}

	go func(ctx *devspacecontext.Context) {
		select {
		case <-ctx.Context.Done():
			<-t.Dead()
			close(d.done)
			return
		case <-t.Dead():
			// try restarting the dev pod if it has stopped because of
			// a lost connection
			if _, ok := t.Err().(DevPodLostConnection); ok {
				for {
					err := d.startWithRetry(ctx, devPodConfig, options)
					if err != nil {
						if ctx.IsDone() {
							return
						}

						ctx.Log.Infof("Restart dev %s because of: %v", devPodConfig.Name, err)
						select {
						case <-ctx.Context.Done():
							return
						case <-time.After(time.Second * 10):
							continue
						}
					}

					return
				}
			} else {
				close(d.done)
			}
		}
	}(ctx)

	// Create a new tomb and run it
	tombCtx := t.Context(ctx.Context)
	ctx = ctx.WithContext(tombCtx)
	var (
		hasTerminal bool
		err         error
	)
	<-t.NotifyGo(func() error {
		hasTerminal, err = d.start(ctx, devPodConfig, options, t)
		return err
	})
	if hasTerminal {
		err = t.Wait()
		if err != nil {
			// if we just lost connection we wait here until stopped
			if _, ok := t.Err().(DevPodLostConnection); ok {
				<-d.done
				return nil
			}

			return err
		}
		return nil
	} else if !t.Alive() {
		return t.Err()
	}

	return nil
}

func (d *devPod) start(ctx *devspacecontext.Context, devPodConfig *latest.DevPod, opts Options, parent *tomb.Tomb) (hasTerminal bool, err error) {
	// check first if we need to replace the pod
	if !opts.DisablePodReplace && needPodReplace(devPodConfig) {
		err := podreplace.NewPodReplacer().ReplacePod(ctx, devPodConfig)
		if err != nil {
			return false, errors.Wrap(err, "replace pod")
		}
	} else {
		devPodCache, ok := ctx.Config.RemoteCache().GetDevPod(devPodConfig.Name)
		if ok && devPodCache.ReplicaSet != "" {
			_, err := podreplace.NewPodReplacer().RevertReplacePod(ctx, &devPodCache)
			if err != nil {
				return false, errors.Wrap(err, "replace pod")
			}
		}
	}

	var imageSelector []string
	if devPodConfig.ImageSelector != "" {
		imageSelectorObject, err := runtimevar.NewRuntimeResolver(ctx.WorkingDir, true).FillRuntimeVariablesAsImageSelector(ctx.Context, devPodConfig.ImageSelector, ctx.Config, ctx.Dependencies)
		if err != nil {
			return false, err
		}

		imageSelector = []string{imageSelectorObject.Image}
	}

	// wait for pod to be ready
	ctx.Log.Infof("Waiting for pod to become ready...")
	options := targetselector.NewEmptyOptions().
		ApplyConfigParameter("", devPodConfig.LabelSelector, imageSelector, devPodConfig.Namespace, "").
		WithWaitingStrategy(targetselector.NewUntilNewestRunningWaitingStrategy(time.Millisecond * 500)).
		WithSkipInitContainers(true)
	d.selectedPod, err = targetselector.NewTargetSelector(options).SelectSinglePod(ctx.Context, ctx.KubeClient, ctx.Log)
	if err != nil {
		return false, errors.Wrap(err, "waiting for pod to become ready")
	}

	// start sync and port forwarding
	err = d.startSyncAndPortForwarding(ctx, devPodConfig, newTargetSelector(d.selectedPod.Name, d.selectedPod.Namespace, parent), opts, parent)
	if err != nil {
		return false, err
	}

	// start logs or terminal
	terminalDevContainer := d.getTerminalDevContainer(devPodConfig)
	if terminalDevContainer != nil {
		return true, d.startTerminal(ctx, terminalDevContainer, parent)
	}

	// TODO attach
	return false, d.startLogs(ctx, devPodConfig, parent)
}

func (d *devPod) startLogs(ctx *devspacecontext.Context, devPodConfig *latest.DevPod, parent *tomb.Tomb) error {
	loader.EachDevContainer(devPodConfig, func(devContainer *latest.DevContainer) bool {
		if devContainer.Logs == nil || devContainer.Logs.Disabled {
			return true
		}

		parent.Go(func() error {
			return logs.StartLogs(ctx, devContainer, newTargetSelector(d.selectedPod.Name, d.selectedPod.Namespace, parent))
		})

		return true
	})

	return nil
}

func (d *devPod) getTerminalDevContainer(devPodConfig *latest.DevPod) *latest.DevContainer {
	// find dev container config
	var devContainer *latest.DevContainer
	if devPodConfig.Terminal != nil && !devPodConfig.Terminal.Disabled {
		devContainer = &devPodConfig.DevContainer
	}
	for _, d := range devPodConfig.Containers {
		if d.Terminal != nil && !d.Terminal.Disabled {
			devContainer = &d
			break
		}
	}

	return devContainer
}

func (d *devPod) startTerminal(ctx *devspacecontext.Context, devContainer *latest.DevContainer, parent *tomb.Tomb) error {
	parent.Go(func() error {
		// make sure the global log is silent
		before := log.GetInstance().GetLevel()
		log.GetInstance().SetLevel(logrus.FatalLevel)
		err := terminal.StartTerminal(ctx, devContainer, newTargetSelector(d.selectedPod.Name, d.selectedPod.Namespace, parent), os.Stdout, os.Stderr, os.Stdin, parent)
		log.GetInstance().SetLevel(before)
		if err != nil {
			return errors.Wrap(err, "error in terminal forwarding")
		}

		// kill ourselves here
		parent.Kill(nil)
		return nil
	})

	return nil
}

func (d *devPod) startSyncAndPortForwarding(ctx *devspacecontext.Context, devPod *latest.DevPod, selector targetselector.TargetSelector, opts Options, parent *tomb.Tomb) error {
	pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{}, "devCommand:before:sync", "dev.beforeSync", "devCommand:before:portForwarding", "dev.beforePortForwarding")
	if pluginErr != nil {
		return pluginErr
	}

	// Start sync
	syncDone := parent.NotifyGo(func() error {
		if opts.DisableSync {
			return nil
		}

		return sync.StartSync(ctx, devPod, selector, parent)
	})

	// Start Port Forwarding
	portForwardingDone := parent.NotifyGo(func() error {
		if opts.DisablePortForwarding {
			return nil
		}

		return portforwarding.StartPortForwarding(ctx, devPod, selector, parent)
	})

	// wait for both to finish
	<-syncDone
	<-portForwardingDone

	// execute hooks
	pluginErr = hook.ExecuteHooks(ctx, map[string]interface{}{}, "devCommand:after:sync", "dev.afterSync", "devCommand:after:portForwarding", "dev.afterPortForwarding")
	if pluginErr != nil {
		return pluginErr
	}

	return nil
}

func needPodReplace(devPodConfig *latest.DevPod) bool {
	if len(devPodConfig.Patches) > 0 {
		return true
	}

	needReplace := false
	loader.EachDevContainer(devPodConfig, func(devContainer *latest.DevContainer) bool {
		if needPodReplaceContainer(&devPodConfig.DevContainer) {
			needReplace = true
			return false
		}
		return true
	})

	return needReplace
}

func needPodReplaceContainer(devContainer *latest.DevContainer) bool {
	if devContainer.ReplaceImage != "" {
		return true
	}
	if len(devContainer.PersistPaths) > 0 {
		return true
	}
	if devContainer.Terminal != nil && !devContainer.Terminal.Disabled && !devContainer.Terminal.DisableReplace {
		return true
	}

	return false
}
