package devpod

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/devspace/services/logs"
	"github.com/loft-sh/devspace/pkg/devspace/services/terminal"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/skratchdot/open-golang/open"
	"io"
	"net/http"
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

var (
	openMaxWait = 5 * time.Minute

	terminalDevPodMutex syncpkg.Mutex
	terminalDevPod      *devPod
)

var (
	DefaultTerminalStdout io.Writer = os.Stdout
	DefaultTerminalStderr io.Writer = os.Stderr
	DefaultTerminalStdin  io.Reader = os.Stdin
)

type devPod struct {
	selectedPod *selector.SelectedPodContainer

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
	<-t.NotifyGo(func() error {
		return d.start(ctx, devPodConfig, options, t)
	})
	if !t.Alive() {
		return t.Err()
	}

	return nil
}

func (d *devPod) start(ctx *devspacecontext.Context, devPodConfig *latest.DevPod, opts Options, parent *tomb.Tomb) error {
	// check first if we need to replace the pod
	if !opts.DisablePodReplace && needPodReplace(devPodConfig) {
		err := podreplace.NewPodReplacer().ReplacePod(ctx, devPodConfig)
		if err != nil {
			return errors.Wrap(err, "replace pod")
		}
	} else {
		devPodCache, ok := ctx.Config.RemoteCache().GetDevPod(devPodConfig.Name)
		if ok && devPodCache.Deployment != "" {
			_, err := podreplace.NewPodReplacer().RevertReplacePod(ctx, &devPodCache)
			if err != nil {
				return errors.Wrap(err, "replace pod")
			}
		}
	}

	var imageSelector []string
	if devPodConfig.ImageSelector != "" {
		imageSelectorObject, err := runtimevar.NewRuntimeResolver(ctx.WorkingDir, true).FillRuntimeVariablesAsImageSelector(ctx.Context, devPodConfig.ImageSelector, ctx.Config, ctx.Dependencies)
		if err != nil {
			return err
		}

		imageSelector = []string{imageSelectorObject.Image}
	}

	// wait for pod to be ready
	ctx.Log.Infof("Waiting for pod to become ready...")
	options := targetselector.NewEmptyOptions().
		ApplyConfigParameter("", devPodConfig.LabelSelector, imageSelector, devPodConfig.Namespace, "").
		WithWaitingStrategy(targetselector.NewUntilNewestRunningWaitingStrategy(time.Millisecond * 500)).
		WithSkipInitContainers(true)
	var err error
	d.selectedPod, err = targetselector.NewTargetSelector(options).SelectSingleContainer(ctx.Context, ctx.KubeClient, ctx.Log)
	if err != nil {
		return errors.Wrap(err, "waiting for pod to become ready")
	}
	ctx.Log.Debugf("Selected pod:container %s:%s", d.selectedPod.Pod.Name, d.selectedPod.Container.Name)

	// Run dev.open configs
	for _, openConfig := range devPodConfig.Open {
		if openConfig.URL != "" {
			url := openConfig.URL
			ctx.Log.Infof("Opening '%s' as soon as application will be started", openConfig.URL)
			parent.Go(func() error {
				now := time.Now()
				for time.Since(now) < openMaxWait {
					select {
					case <-ctx.Context.Done():
						return nil
					case <-time.After(time.Second):
						resp, _ := http.Get(url)
						if resp != nil && resp.StatusCode != http.StatusBadGateway && resp.StatusCode != http.StatusServiceUnavailable {
							time.Sleep(time.Second * 1)
							_ = open.Start(url)
							ctx.Log.Donef("Successfully opened %s", url)
						}
					}
				}

				return nil
			})
		}
	}

	// start sync and port forwarding
	err = d.startSyncAndPortForwarding(ctx, devPodConfig, newTargetSelector(d.selectedPod.Pod.Name, d.selectedPod.Pod.Namespace, d.selectedPod.Container.Name, parent), opts, parent)
	if err != nil {
		return err
	}

	// start logs or terminal
	terminalDevContainer := d.getTerminalDevContainer(devPodConfig)
	if terminalDevContainer != nil {
		return d.startTerminal(ctx, terminalDevContainer, opts, parent)
	}

	// TODO attach
	return d.startLogs(ctx, devPodConfig, parent)
}

func (d *devPod) startLogs(ctx *devspacecontext.Context, devPodConfig *latest.DevPod, parent *tomb.Tomb) error {
	loader.EachDevContainer(devPodConfig, func(devContainer *latest.DevContainer) bool {
		if devContainer.Logs == nil || devContainer.Logs.Disabled {
			return true
		}

		parent.Go(func() error {
			return logs.StartLogs(ctx, devContainer, newTargetSelector(d.selectedPod.Pod.Name, d.selectedPod.Pod.Namespace, d.selectedPod.Container.Name, parent))
		})

		return true
	})

	return nil
}

func (d *devPod) getTerminalDevContainer(devPodConfig *latest.DevPod) *latest.DevContainer {
	// find dev container config
	var devContainer *latest.DevContainer
	loader.EachDevContainer(devPodConfig, func(d *latest.DevContainer) bool {
		if d.Terminal != nil && !d.Terminal.Disabled {
			devContainer = d
			return false
		}
		return true
	})

	return devContainer
}

func (d *devPod) startTerminal(ctx *devspacecontext.Context, devContainer *latest.DevContainer, opts Options, parent *tomb.Tomb) error {
	parent.Go(func() error {
		err := setTerminalDevPod(d)
		if err != nil {
			return err
		}

		// make sure the global log is silent
		err = terminal.StartTerminal(
			ctx,
			devContainer,
			newTargetSelector(d.selectedPod.Pod.Name, d.selectedPod.Pod.Namespace, d.selectedPod.Container.Name, parent),
			DefaultTerminalStdout,
			DefaultTerminalStderr,
			DefaultTerminalStdin,
			parent,
		)
		terminalDevPodMutex.Lock()
		terminalDevPod = nil
		terminalDevPodMutex.Unlock()
		if err != nil {
			return errors.Wrap(err, "error in terminal forwarding")
		}

		// kill ourselves here
		if !opts.ContinueOnTerminalExit && opts.KillApplication != nil {
			go opts.KillApplication()
		} else {
			parent.Kill(nil)
		}
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
	if len(devContainer.Env) > 0 {
		return true
	}
	if len(devContainer.Command) > 0 {
		return true
	}
	if devContainer.Args != nil {
		return true
	}
	if !devContainer.DisableRestartHelper {
		for _, s := range devContainer.Sync {
			if s.OnUpload != nil && s.OnUpload.RestartContainer {
				return true
			}
		}
	}

	return false
}

func setTerminalDevPod(devPod *devPod) error {
	terminalDevPodMutex.Lock()
	defer terminalDevPodMutex.Unlock()

	if terminalDevPod != nil {
		return fmt.Errorf("error starting terminal as it is currently already used by another dev pod")
	}

	terminalDevPod = devPod
	return nil
}
