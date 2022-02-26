package devpod

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/devspace/services/logs"
	"github.com/loft-sh/devspace/pkg/devspace/services/terminal"
	"github.com/loft-sh/devspace/pkg/util/log"
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
	Start(ctx *devspacecontext.Context, devPodConfig *latest.DevPod) error
	Stop()
	Done() <-chan struct{}
	Error() error
}

type devPod struct {
	ctxMutex syncpkg.Mutex

	devCtx        *devspacecontext.Context
	imageSelector []string
	devPodConfig  *latest.DevPod
	cancel        context.CancelFunc
	exitErr       error
	selectedPod   *corev1.Pod

	done chan struct{}
}

func newDevPod() *devPod {
	return &devPod{}
}

func (d *devPod) Start(ctx *devspacecontext.Context, devPodConfig *latest.DevPod) error {
	d.ctxMutex.Lock()
	defer d.ctxMutex.Unlock()
	if d.devCtx != nil {
		return errors.Errorf("dev pod is already running, please stop it before starting")
	}

	var devCtx context.Context
	devCtx, d.cancel = context.WithCancel(ctx.Context)
	ctx = ctx.WithContext(devCtx)

	d.devCtx = ctx
	d.devPodConfig = devPodConfig
	err := d.start(ctx, devPodConfig)
	if err != nil {
		d.stop(err)
		return errors.Wrap(err, "starting dev pod")
	}

	return nil
}

func (d *devPod) Error() error {
	d.ctxMutex.Lock()
	defer d.ctxMutex.Unlock()
	return d.exitErr
}

func (d *devPod) Stop() {
	d.ctxMutex.Lock()
	defer d.ctxMutex.Unlock()

	d.stop(nil)
}

func (d *devPod) Done() <-chan struct{} {
	d.ctxMutex.Lock()
	defer d.ctxMutex.Unlock()

	if d.devCtx != nil {
		return d.devCtx.Context.Done()
	}
	return nil
}

func (d *devPod) stop(withErr error) {
	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
		d.exitErr = withErr

		if d.done != nil {
			<-d.done
		}
	}
}

func (d *devPod) start(ctx *devspacecontext.Context, devPodConfig *latest.DevPod) (err error) {
	// check first if we need to replace the pod
	if needPodReplace(devPodConfig) {
		err := podreplace.NewPodReplacer().ReplacePod(ctx, devPodConfig)
		if err != nil {
			return errors.Wrap(err, "replace pod")
		}
	} else {
		devPodCache, ok := ctx.Config.RemoteCache().GetDevPod(devPodConfig.Name)
		if ok {
			_, err := podreplace.NewPodReplacer().RevertReplacePod(ctx, &devPodCache)
			if err != nil {
				return errors.Wrap(err, "replace pod")
			}
		}
	}

	if d.devPodConfig.ImageSelector != "" {
		imageSelectorObject, err := runtimevar.NewRuntimeResolver(ctx.WorkingDir, true).FillRuntimeVariablesAsImageSelector(d.devPodConfig.ImageSelector, d.devCtx.Config, d.devCtx.Dependencies)
		if err != nil {
			return err
		}

		d.imageSelector = []string{imageSelectorObject.Image}
	}

	// wait for pod to be ready
	ctx.Log.Infof("Waiting for pod to become ready...")
	d.selectedPod, err = targetselector.NewTargetSelector(d.newOptions("")).SelectSinglePod(ctx.Context, ctx.KubeClient, ctx.Log)
	if err != nil {
		return errors.Wrap(err, "waiting for pod to become ready")
	}

	// start sync and port forwarding
	d.done = make(chan struct{})
	err = d.startSyncAndPortForwarding(ctx, devPodConfig, &targetSelectorWithContainer{devPod: d}, d.done)
	if err != nil {
		return err
	}

	// start logs or terminal
	hasTerminal := d.startTerminal(ctx, devPodConfig)
	if !hasTerminal {
		return d.startLogs(ctx, devPodConfig)
	}

	// TODO attach
	return nil
}

func (d *devPod) startLogs(ctx *devspacecontext.Context, devPodConfig *latest.DevPod) error {
	if devPodConfig.Logs != nil && !devPodConfig.Logs.Disabled {
		go func() {
			err := logs.StartLogs(ctx, &devPodConfig.DevContainer, &targetSelectorWithContainer{devPod: d})
			if err != nil {
				ctx.Log.Errorf("error in printing logs: %v", err)
			}
		}()
	}
	for _, c := range devPodConfig.Containers {
		if c.Logs != nil && !c.Logs.Disabled {
			go func() {
				err := logs.StartLogs(ctx, &devPodConfig.DevContainer, &targetSelectorWithContainer{devPod: d})
				if err != nil {
					ctx.Log.Errorf("error in printing logs: %v", err)
				}
			}()
		}
	}
	return nil
}

func (d *devPod) startTerminal(ctx *devspacecontext.Context, devPodConfig *latest.DevPod) bool {
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

	if devContainer != nil {
		_, err := terminal.StartTerminal(ctx, devContainer, &targetSelectorWithContainer{devPod: d}, os.Stdout, os.Stderr, os.Stdin)
		if err != nil {
			ctx.Log.Errorf("error in terminal forwarding: %v", err)
		}

		return true
	}

	return false
}

func (d *devPod) newOptions(container string) targetselector.Options {
	return targetselector.NewEmptyOptions().
		ApplyConfigParameter(container, d.devPodConfig.LabelSelector, d.imageSelector, d.devPodConfig.Namespace, "").
		WithWaitingStrategy(targetselector.NewUntilNewestRunningWaitingStrategy(time.Millisecond * 500)).
		WithSkipInitContainers(true)
}

func (d *devPod) selectSinglePod(ctx context.Context, client kubectl.Client, log log.Logger) (*corev1.Pod, error) {
	pod, err := targetselector.NewTargetSelector(d.newOptions("")).SelectSinglePod(ctx, client, log)
	if err != nil {
		return nil, err
	} else if pod.Namespace != d.selectedPod.Namespace || pod.Name != d.selectedPod.Name {
		d.ctxMutex.Lock()
		defer d.ctxMutex.Unlock()

		go d.stop(errors.Wrap(err, "select pod"))
		return nil, fmt.Errorf("selected pod %s/%s differs from initially selected pod %s/%s", pod.Namespace, pod.Name, d.selectedPod.Namespace, d.selectedPod.Name)
	}

	return pod, nil
}

func (d *devPod) selectSingleWithContainer(ctx context.Context, client kubectl.Client, container string, log log.Logger) (*selector.SelectedPodContainer, error) {
	selectedContainer, err := targetselector.NewTargetSelector(d.newOptions(container)).SelectSingleContainer(ctx, client, log)
	if err != nil {
		return nil, err
	} else if selectedContainer.Pod.Namespace != d.selectedPod.Namespace || selectedContainer.Pod.Name != d.selectedPod.Name {
		d.ctxMutex.Lock()
		defer d.ctxMutex.Unlock()

		go d.stop(errors.Wrap(err, "select pod container"))
		return nil, fmt.Errorf("selected pod %s/%s differs from initially selected pod %s/%s", selectedContainer.Pod.Namespace, selectedContainer.Pod.Name, d.selectedPod.Namespace, d.selectedPod.Name)
	}

	return selectedContainer, nil
}

type targetSelectorWithContainer struct {
	devPod    *devPod
	container string
}

func (d *targetSelectorWithContainer) SelectSinglePod(ctx context.Context, client kubectl.Client, log log.Logger) (*corev1.Pod, error) {
	return d.devPod.selectSinglePod(ctx, client, log)
}

func (d *targetSelectorWithContainer) SelectSingleContainer(ctx context.Context, client kubectl.Client, log log.Logger) (*selector.SelectedPodContainer, error) {
	return d.devPod.selectSingleWithContainer(ctx, client, d.container, log)
}

func (d *targetSelectorWithContainer) WithContainer(container string) targetselector.TargetSelector {
	return &targetSelectorWithContainer{
		devPod:    d.devPod,
		container: container,
	}
}

func (d *devPod) startSyncAndPortForwarding(ctx *devspacecontext.Context, devPod *latest.DevPod, selector targetselector.TargetSelector, done chan struct{}) error {
	errChan := make(chan error, 2)
	pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{}, "devCommand:before:sync", "dev.beforeSync", "devCommand:before:portForwarding", "dev.beforePortForwarding")
	if pluginErr != nil {
		return pluginErr
	}

	// Start sync
	syncDone := make(chan struct{})
	go func() {
		// start sync
		err := sync.StartSync(ctx, devPod, selector, syncDone)
		if err != nil {
			errChan <- errors.Wrap(err, "start sync")
			return
		}

		errChan <- nil
	}()

	// Start Port Forwarding
	portForwardingDone := make(chan struct{})
	go func() {
		// start port forwarding
		err := portforwarding.StartPortForwarding(ctx, devPod, selector, portForwardingDone)
		if err != nil {
			errChan <- errors.Errorf("Unable to start portforwarding: %v", err)
			return
		}

		errChan <- nil
	}()

	// make sure we close the done channel correctly
	go func() {
		<-syncDone
		<-portForwardingDone
		close(done)
	}()

	// wait for sync and port forwarding
	for i := 0; i < 2; i++ {
		err := <-errChan
		if err != nil {
			return err
		}
	}

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

type DevPodOverrideError struct{}

func (n DevPodOverrideError) Error() string {
	return "there is a newer dev pod attached"
}

type DevPodMismatchError struct{}

func (n DevPodMismatchError) Error() string {
	return "another pod has matched "
}
