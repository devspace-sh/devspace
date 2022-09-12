package devpod

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/kill"
	"io"
	"net/http"
	"os"
	syncpkg "sync"

	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/mgutz/ansi"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/devspace/services/attach"
	"github.com/loft-sh/devspace/pkg/devspace/services/logs"
	"github.com/loft-sh/devspace/pkg/devspace/services/proxycommands"
	"github.com/loft-sh/devspace/pkg/devspace/services/ssh"
	"github.com/loft-sh/devspace/pkg/devspace/services/terminal"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
	"gopkg.in/yaml.v3"

	"time"

	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/services/podreplace"
	"github.com/loft-sh/devspace/pkg/devspace/services/portforwarding"
	"github.com/loft-sh/devspace/pkg/devspace/services/sync"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/pkg/errors"
)

var (
	openMaxWait = 5 * time.Minute
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
	err  error

	cancelCtx context.Context
	cancel    context.CancelFunc
}

func newDevPod() *devPod {
	return &devPod{
		done: make(chan struct{}),
	}
}

func (d *devPod) Start(ctx devspacecontext.Context, devPodConfig *latest.DevPod, options Options) error {
	d.m.Lock()
	if d.cancel != nil {
		d.m.Unlock()
		return errors.Errorf("dev pod is already running, please stop it before starting")
	}

	d.cancelCtx, d.cancel = context.WithCancel(ctx.Context())
	ctx = ctx.WithContext(d.cancelCtx)
	d.m.Unlock()

	// log devpod to console if debug
	if ctx.Log().GetLevel() == logrus.DebugLevel {
		out, err := yaml.Marshal(devPodConfig)
		if err == nil {
			ctx.Log().Debugf("DevPod Config: \n%s\n", string(out))
		}
	}

	// start the dev pod
	err := d.startWithRetry(ctx, devPodConfig, options)
	if err != nil {
		d.Stop()
		return err
	}

	return nil
}

func (d *devPod) Err() error {
	d.m.Lock()
	defer d.m.Unlock()

	return d.err
}

func (d *devPod) Done() <-chan struct{} {
	return d.done
}

func (d *devPod) Stop() {
	d.m.Lock()
	if d.cancel != nil {
		d.cancel()
	}
	d.m.Unlock()
	<-d.done
}

func (d *devPod) startWithRetry(ctx devspacecontext.Context, devPodConfig *latest.DevPod, options Options) error {
	t := &tomb.Tomb{}

	go func(ctx devspacecontext.Context) {
		// wait for parent context cancel
		// or that the DevPod is done
		select {
		case <-ctx.Context().Done():
		case <-t.Dead():
		}

		if ctx.IsDone() {
			<-t.Dead()
			ctx.Log().Debugf("Stopped dev %s", devPodConfig.Name)
			close(d.done)
			return
		}

		// check if pod was terminated
		d.m.Lock()
		selectedPod := d.selectedPod
		d.selectedPod = nil
		d.m.Unlock()

		// check if we need to restart
		if selectedPod != nil {
			shouldRestart := false
			err := wait.PollImmediateUntil(time.Second, func() (bool, error) {
				pod, err := ctx.KubeClient().KubeClient().CoreV1().Pods(selectedPod.Pod.Namespace).Get(ctx.Context(), selectedPod.Pod.Name, metav1.GetOptions{})
				if err != nil {
					if kerrors.IsNotFound(err) {
						ctx.Log().Debugf("Restart dev %s because pod isn't found anymore", devPodConfig.Name)
						shouldRestart = true
						return true, nil
					}

					// this case means there might be problems with internet
					ctx.Log().Debugf("error trying to retrieve pod: %v", err)
					return false, nil
				} else if pod.DeletionTimestamp != nil {
					ctx.Log().Debugf("Restart dev %s because pod is terminating", devPodConfig.Name)
					shouldRestart = true
					return true, nil
				}

				return true, nil
			}, ctx.Context().Done())
			if err != nil {
				if err != wait.ErrWaitTimeout {
					ctx.Log().Errorf("error restarting dev: %v", err)
				}
			} else if shouldRestart {
				d.restart(ctx, devPodConfig, options)
				return
			}
		}

		ctx.Log().Debugf("Stopped dev %s", devPodConfig.Name)
		d.m.Lock()
		d.err = t.Err()
		d.m.Unlock()
		close(d.done)
	}(ctx)

	// Create a new tomb and run it
	tombCtx := t.Context(ctx.Context())
	ctx = ctx.WithContext(tombCtx)
	<-t.NotifyGo(func() error {
		return d.start(ctx, devPodConfig, options, t)
	})
	if !t.Alive() {
		return t.Err()
	}

	return nil
}

func (d *devPod) restart(ctx devspacecontext.Context, devPodConfig *latest.DevPod, options Options) {
	for {
		err := d.startWithRetry(ctx, devPodConfig, options)
		if err != nil {
			if ctx.IsDone() {
				return
			}

			ctx.Log().Infof("Restart dev %s because of: %v", devPodConfig.Name, err)
			select {
			case <-ctx.Context().Done():
				return
			case <-time.After(time.Second * 10):
				continue
			}
		}

		return
	}
}

func (d *devPod) start(ctx devspacecontext.Context, devPodConfig *latest.DevPod, opts Options, parent *tomb.Tomb) error {
	// check first if we need to replace the pod
	if !opts.DisablePodReplace && needPodReplace(devPodConfig) {
		err := podreplace.NewPodReplacer().ReplacePod(ctx, devPodConfig)
		if err != nil {
			return errors.Wrap(err, "replace pod")
		}
	} else {
		devPodCache, ok := ctx.Config().RemoteCache().GetDevPod(devPodConfig.Name)
		if ok && devPodCache.Deployment != "" {
			_, err := podreplace.NewPodReplacer().RevertReplacePod(ctx, &devPodCache, &deploy.PurgeOptions{ForcePurge: true})
			if err != nil {
				return errors.Wrap(err, "replace pod")
			}
		}
	}

	var imageSelector []string
	if devPodConfig.ImageSelector != "" {
		imageSelectorObject, err := runtimevar.NewRuntimeResolver(ctx.WorkingDir(), true).FillRuntimeVariablesAsImageSelector(ctx.Context(), devPodConfig.ImageSelector, ctx.Config(), ctx.Dependencies())
		if err != nil {
			return err
		}

		imageSelector = []string{imageSelectorObject.Image}
	}

	// wait for pod to be ready
	ctx.Log().Infof("Waiting for pod to become ready...")
	options := targetselector.NewEmptyOptions().
		ApplyConfigParameter("", devPodConfig.LabelSelector, imageSelector, devPodConfig.Namespace, "").
		WithWaitingStrategy(targetselector.NewUntilNewestRunningWaitingStrategy(time.Millisecond * 500)).
		WithSkipInitContainers(true)
	var err error
	selectedPod, err := targetselector.NewTargetSelector(options).SelectSingleContainer(ctx.Context(), ctx.KubeClient(), ctx.Log())
	if err != nil {
		return errors.Wrap(err, "waiting for pod to become ready")
	}

	// check if the correct pod is matched
	loader.EachDevContainer(devPodConfig, func(devContainer *latest.DevContainer) bool {
		if devContainer.Container == "" {
			return true
		}

		// check if the container exists in the pod
		for _, container := range selectedPod.Pod.Spec.Containers {
			if container.Name == devContainer.Container {
				return true
			}
		}
		for _, container := range selectedPod.Pod.Spec.InitContainers {
			if container.Name == devContainer.Container {
				return true
			}
		}

		err = fmt.Errorf("selected pod '%s/%s' doesn't include container '%s', please make sure you don't have overlapping label selectors within the namespace and the pod you select contains container '%s'", selectedPod.Pod.Namespace, selectedPod.Pod.Name, devContainer.Container, devContainer.Container)
		return false
	})
	if err != nil {
		return errors.Wrap(err, "select pod")
	}
	ctx.Log().Infof("Selected pod %s", ansi.Color(selectedPod.Pod.Name, "yellow+b"))

	// set selected pod
	d.m.Lock()
	d.selectedPod = selectedPod
	d.m.Unlock()

	// Run dev.open configs
	if !opts.DisableOpen {
		ctx := ctx.WithLogger(ctx.Log().WithPrefixColor("open  ", "yellow+b"))
		for _, openConfig := range devPodConfig.Open {
			if openConfig.URL != "" {
				url := openConfig.URL
				ctx.Log().Infof("Opening '%s' as soon as application will be started", openConfig.URL)
				parent.Go(func() error {
					now := time.Now()
					for time.Since(now) < openMaxWait {
						select {
						case <-ctx.Context().Done():
							return nil
						case <-time.After(time.Second):
							err := tryOpen(ctx.Context(), url, ctx.Log())
							if err == nil {
								return nil
							}
						}
					}

					return nil
				})
			}
		}
	}

	// start sync and port forwarding
	err = d.startServices(ctx, devPodConfig, newTargetSelector(selectedPod.Pod.Name, selectedPod.Pod.Namespace, selectedPod.Container.Name, parent), opts, parent)
	if err != nil {
		return err
	}

	// start logs
	terminalDevContainer := d.getTerminalDevContainer(devPodConfig)
	if terminalDevContainer != nil {
		return d.startTerminal(ctx, terminalDevContainer, opts, selectedPod, parent)
	}

	// start attach if defined
	attachDevContainer := d.getAttachDevContainer(devPodConfig)
	if attachDevContainer != nil {
		return d.startAttach(ctx, attachDevContainer, opts, selectedPod, parent)
	}

	return d.startLogs(ctx, devPodConfig, selectedPod, parent)
}

func tryOpen(ctx context.Context, url string, log logpkg.Logger) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, "GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()

	if resp != nil && resp.StatusCode != http.StatusBadGateway && resp.StatusCode != http.StatusServiceUnavailable {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
		}
		_ = open.Start(url)
		log.Donef("Successfully opened %s", url)
		return nil
	}

	return fmt.Errorf("not reachable")
}

func (d *devPod) startLogs(ctx devspacecontext.Context, devPodConfig *latest.DevPod, selectedPod *selector.SelectedPodContainer, parent *tomb.Tomb) error {
	ctx = ctx.WithLogger(ctx.Log().WithPrefixColor("logs  ", "yellow+b"))
	loader.EachDevContainer(devPodConfig, func(devContainer *latest.DevContainer) bool {
		if devContainer.Logs == nil || (devContainer.Logs.Enabled != nil && !*devContainer.Logs.Enabled) {
			return true
		}

		parent.Go(func() error {
			return logs.StartLogs(ctx, devContainer, newTargetSelector(selectedPod.Pod.Name, selectedPod.Pod.Namespace, selectedPod.Container.Name, parent))
		})

		return true
	})

	return nil
}

func (d *devPod) getAttachDevContainer(devPodConfig *latest.DevPod) *latest.DevContainer {
	// find dev container config
	var devContainer *latest.DevContainer
	loader.EachDevContainer(devPodConfig, func(d *latest.DevContainer) bool {
		if d.Attach == nil || (d.Attach.Enabled != nil && !*d.Attach.Enabled) {
			return true
		}
		devContainer = d
		return false
	})

	return devContainer
}

func (d *devPod) getTerminalDevContainer(devPodConfig *latest.DevPod) *latest.DevContainer {
	// find dev container config
	var devContainer *latest.DevContainer
	loader.EachDevContainer(devPodConfig, func(d *latest.DevContainer) bool {
		if d.Terminal == nil || (d.Terminal.Enabled != nil && !*d.Terminal.Enabled) {
			return true
		}
		devContainer = d
		return false
	})

	return devContainer
}

func (d *devPod) startAttach(ctx devspacecontext.Context, devContainer *latest.DevContainer, opts Options, selectedPod *selector.SelectedPodContainer, parent *tomb.Tomb) error {
	parent.Go(func() error {
		id, err := logpkg.AcquireGlobalSilence()
		if err != nil {
			return err
		}
		defer logpkg.ReleaseGlobalSilence(id)

		// make sure the global log is silent
		ctx = ctx.WithLogger(ctx.Log().WithPrefixColor("attach ", "yellow+b"))
		err = attach.StartAttach(
			ctx,
			devContainer,
			newTargetSelector(selectedPod.Pod.Name, selectedPod.Pod.Namespace, selectedPod.Container.Name, parent),
			DefaultTerminalStdout,
			DefaultTerminalStderr,
			DefaultTerminalStdin,
			parent,
		)
		if err != nil {
			return errors.Wrap(err, "error in attach")
		}

		// if context is done we just return
		if ctx.IsDone() {
			return nil
		}

		// kill ourselves here
		if !opts.ContinueOnTerminalExit {
			kill.StopDevSpace("")
		} else {
			parent.Kill(nil)
		}
		return nil
	})

	return nil
}

func (d *devPod) startTerminal(ctx devspacecontext.Context, devContainer *latest.DevContainer, opts Options, selectedPod *selector.SelectedPodContainer, parent *tomb.Tomb) error {
	parent.Go(func() error {
		id, err := logpkg.AcquireGlobalSilence()
		if err != nil {
			return err
		}
		defer logpkg.ReleaseGlobalSilence(id)

		// make sure the global log is silent
		ctx = ctx.WithLogger(ctx.Log().WithPrefixColor("term  ", "yellow+b"))
		err = terminal.StartTerminal(
			ctx,
			devContainer,
			newTargetSelector(selectedPod.Pod.Name, selectedPod.Pod.Namespace, selectedPod.Container.Name, parent),
			DefaultTerminalStdout,
			DefaultTerminalStderr,
			DefaultTerminalStdin,
			parent,
		)
		if err != nil {
			return errors.Wrap(err, "error in terminal forwarding")
		}

		// if context is done we just return
		if ctx.IsDone() {
			return nil
		}

		// kill ourselves here
		if !opts.ContinueOnTerminalExit {
			kill.StopDevSpace("")
		} else {
			parent.Kill(nil)
		}
		return nil
	})

	return nil
}

func (d *devPod) startServices(ctx devspacecontext.Context, devPod *latest.DevPod, selector targetselector.TargetSelector, opts Options, parent *tomb.Tomb) error {
	pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{}, "devCommand:before:sync", "dev.beforeSync", "devCommand:before:portForwarding", "dev.beforePortForwarding")
	if pluginErr != nil {
		return pluginErr
	}

	// Start sync
	syncDone := parent.NotifyGo(func() error {
		if opts.DisableSync {
			return nil
		}

		// add prefix
		ctx := ctx.WithLogger(ctx.Log().WithPrefixColor("sync  ", "yellow+b"))
		err := sync.StartSync(ctx, devPod, selector, parent)
		return err
	})

	// Start Port Forwarding
	portForwardingDone := parent.NotifyGo(func() error {
		if opts.DisablePortForwarding {
			return nil
		}

		ctx := ctx.WithLogger(ctx.Log().WithPrefixColor("ports ", "yellow+b"))
		return portforwarding.StartPortForwarding(ctx, devPod, selector, parent)
	})

	// wait for both to finish
	<-syncDone
	<-portForwardingDone

	// Start SSH
	sshDone := parent.NotifyGo(func() error {
		// add ssh prefix
		ctx := ctx.WithLogger(ctx.Log().WithPrefixColor("ssh   ", "yellow+b"))
		return ssh.StartSSH(ctx, devPod, selector, parent)
	})

	// Start Reverse Commands
	reverseCommandsDone := parent.NotifyGo(func() error {
		// add proxy prefix
		ctx := ctx.WithLogger(ctx.Log().WithPrefixColor("proxy ", "yellow+b"))
		return proxycommands.StartProxyCommands(ctx, devPod, selector, parent)
	})

	// wait for ssh and reverse commands
	<-sshDone
	<-reverseCommandsDone

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
		if needPodReplaceContainer(devContainer) {
			needReplace = true
			return false
		}
		return true
	})

	return needReplace
}

func needPodReplaceContainer(devContainer *latest.DevContainer) bool {
	if devContainer.DevImage != "" {
		return true
	}
	if len(devContainer.PersistPaths) > 0 {
		return true
	}
	if devContainer.RestartHelper != nil && devContainer.RestartHelper.Inject != nil && *devContainer.RestartHelper.Inject {
		return true
	}
	if devContainer.Terminal != nil && !devContainer.Terminal.DisableReplace && (devContainer.Terminal.Enabled == nil || *devContainer.Terminal.Enabled) {
		return true
	}
	if devContainer.Attach != nil && !devContainer.Attach.DisableReplace && (devContainer.Attach.Enabled == nil || *devContainer.Attach.Enabled) {
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
	if devContainer.RestartHelper == nil || devContainer.RestartHelper.Inject == nil || *devContainer.RestartHelper.Inject {
		for _, s := range devContainer.Sync {
			if s.OnUpload != nil && s.OnUpload.RestartContainer {
				return true
			}
		}
	}
	if devContainer.WorkingDir != "" {
		return true
	}
	if devContainer.Resources != nil {
		return true
	}

	return false
}
