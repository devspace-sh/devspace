package devpod

import (
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

type DevPod interface {
	// Start starts the DevPod with the given configuration
	Start(ctx *devspacecontext.Context, devPodConfig *latest.DevPod) error

	// Stop just closes all connections to the DevPod and waits for things
	// to get cleaned up
	Stop()

	// Done returns a channel that is closed as soon as everything is done
	// and cleaned up
	Done() <-chan struct{}

	// Alive returns if the dev pod is still doing something in the background
	Alive() bool

	// Error returns an error if the DevPod exited because of an error
	Error() error
}

type devPod struct {
	m           syncpkg.Mutex
	started     bool
	selectedPod *corev1.Pod

	job string
	t   *tomb.Tomb
}

func newDevPod() *devPod {
	return &devPod{
		t: &tomb.Tomb{},
	}
}

func (d *devPod) Start(ctx *devspacecontext.Context, devPodConfig *latest.DevPod) error {
	d.m.Lock()
	defer d.m.Unlock()

	if d.started {
		return errors.Errorf("dev pod is already running, please stop it before starting")
	}

	d.started = true

	tombCtx := d.t.Context(ctx.Context)
	ctx = ctx.WithContext(tombCtx)

	// start the dev pod
	<-d.t.NotifyGo(func() error {
		return d.start(ctx, devPodConfig, d.t)
	})

	if !d.t.Alive() {
		return d.t.Err()
	}
	return nil
}

func (d *devPod) Alive() bool {
	return d.t.Alive()
}

func (d *devPod) Err() error {
	return d.t.Err()
}

func (d *devPod) Stop() {
	d.t.Kill(nil)
	_ = d.t.Wait()
}

func (d *devPod) Done() <-chan struct{} {
	return d.t.Dead()
}

func (d *devPod) start(ctx *devspacecontext.Context, devPodConfig *latest.DevPod, parent *tomb.Tomb) (err error) {
	// check first if we need to replace the pod
	if needPodReplace(devPodConfig) {
		err := podreplace.NewPodReplacer().ReplacePod(ctx, devPodConfig)
		if err != nil {
			return errors.Wrap(err, "replace pod")
		}
	} else {
		devPodCache, ok := ctx.Config.RemoteCache().GetDevPod(devPodConfig.Name)
		if ok && devPodCache.ReplicaSet != "" {
			_, err := podreplace.NewPodReplacer().RevertReplacePod(ctx, &devPodCache)
			if err != nil {
				return errors.Wrap(err, "replace pod")
			}
		}
	}

	var imageSelector []string
	if devPodConfig.ImageSelector != "" {
		imageSelectorObject, err := runtimevar.NewRuntimeResolver(ctx.WorkingDir, true).FillRuntimeVariablesAsImageSelector(devPodConfig.ImageSelector, ctx.Config, ctx.Dependencies)
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
	d.selectedPod, err = targetselector.NewTargetSelector(options).SelectSinglePod(ctx.Context, ctx.KubeClient, ctx.Log)
	if err != nil {
		return errors.Wrap(err, "waiting for pod to become ready")
	}

	// start sync and port forwarding
	err = d.startSyncAndPortForwarding(ctx, devPodConfig, newTargetSelector(d.selectedPod.Name, d.selectedPod.Namespace, d.t), parent)
	if err != nil {
		return err
	}

	// start logs or terminal
	terminalDevContainer := d.getTerminalDevContainer(devPodConfig)
	if terminalDevContainer != nil {
		return d.startTerminal(ctx, terminalDevContainer, parent)
	}

	// TODO attach
	return d.startLogs(ctx, devPodConfig, parent)
}

func (d *devPod) startLogs(ctx *devspacecontext.Context, devPodConfig *latest.DevPod, parent *tomb.Tomb) error {
	loader.EachDevContainer(devPodConfig, func(devContainer *latest.DevContainer) bool {
		if devContainer.Logs != nil && !devContainer.Logs.Disabled {
			return false
		}

		parent.Go(func() error {
			return logs.StartLogs(ctx, devContainer, newTargetSelector(d.selectedPod.Name, d.selectedPod.Namespace, d.t))
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
	// make sure the global log is silent
	before := log.GetInstance().GetLevel()
	log.GetInstance().SetLevel(logrus.FatalLevel)
	err := terminal.StartTerminal(ctx, devContainer, newTargetSelector(d.selectedPod.Name, d.selectedPod.Namespace, d.t), os.Stdout, os.Stderr, os.Stdin, parent)
	log.GetInstance().SetLevel(before)
	if err != nil {
		return errors.Wrap(err, "error in terminal forwarding")
	}

	return nil
}

func (d *devPod) startSyncAndPortForwarding(ctx *devspacecontext.Context, devPod *latest.DevPod, selector targetselector.TargetSelector, parent *tomb.Tomb) error {
	pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{}, "devCommand:before:sync", "dev.beforeSync", "devCommand:before:portForwarding", "dev.beforePortForwarding")
	if pluginErr != nil {
		return pluginErr
	}

	// Start sync
	syncDone := parent.NotifyGo(func() error {
		return sync.StartSync(ctx, devPod, selector, parent)
	})

	// Start Port Forwarding
	portForwardingDone := parent.NotifyGo(func() error {
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
