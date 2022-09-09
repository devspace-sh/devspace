package devpod

import (
	"context"
	"sync"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/services/podreplace"
	"github.com/loft-sh/devspace/pkg/util/lockfactory"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/stringutil"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

type Options struct {
	ContinueOnTerminalExit bool `long:"continue-on-terminal-exit" description:"Continue on terminal exit"`

	DisableSync           bool `long:"disable-sync" description:"If enabled will not start any sync configuration"`
	DisablePortForwarding bool `long:"disable-port-forwarding" description:"If enabled will not start any port forwarding configuration"`
	DisablePodReplace     bool `long:"disable-pod-replace" description:"If enabled will not replace any pods"`
	DisableOpen           bool `long:"disable-open" description:"If enabled will not replace any pods"`
}

type Manager interface {
	// StartMultiple will start multiple or all dev pods
	StartMultiple(ctx devspacecontext.Context, devPods []string, options Options) error

	// Reset will stop the DevPod if it exists and reset the replaced pods
	Reset(ctx devspacecontext.Context, name string, options *deploy.PurgeOptions) error

	// Stop will stop a specific DevPod
	Stop(ctx devspacecontext.Context, name string)

	// List lists the currently active dev pods
	List() []string

	// Close will close the manager and wait for all dev pods to stop
	Close()

	// Wait will wait until all DevPods are stopped
	Wait() error
}

type devPodManager struct {
	lockFactory lockfactory.LockFactory

	m       sync.Mutex
	cancels []context.CancelFunc
	devPods map[string]*devPod
}

func NewManager(cancel context.CancelFunc) Manager {
	return &devPodManager{
		cancels:     []context.CancelFunc{cancel},
		lockFactory: lockfactory.NewDefaultLockFactory(),
		devPods:     map[string]*devPod{},
	}
}

func (d *devPodManager) List() []string {
	d.m.Lock()
	defer d.m.Unlock()

	retArr := []string{}
	for k := range d.devPods {
		retArr = append(retArr, k)
	}

	return retArr
}

func (d *devPodManager) Close() {
	d.m.Lock()
	for _, cancel := range d.cancels {
		cancel()
	}
	d.cancels = []context.CancelFunc{}
	d.m.Unlock()
	_ = d.Wait()
}

func (d *devPodManager) StartMultiple(ctx devspacecontext.Context, devPods []string, options Options) error {
	devCtx, _ := values.DevContextFrom(ctx.Context())
	select {
	case <-devCtx.Done():
		return devCtx.Err()
	default:
	}

	cancelCtx, cancel := context.WithCancel(devCtx)
	d.m.Lock()
	d.cancels = append(d.cancels, cancel)
	d.m.Unlock()
	ctx = ctx.WithContext(cancelCtx)

	initChans := []chan struct{}{}
	errors := make(chan error, len(ctx.Config().Config().Dev))
	for devPodName, devPod := range ctx.Config().Config().Dev {
		if len(devPods) > 0 && !stringutil.Contains(devPods, devPodName) {
			continue
		}

		initChan := make(chan struct{})
		initChans = append(initChans, initChan)
		go func(devPod *latest.DevPod) {
			defer close(initChan)

			_, err := d.Start(ctx, devPod, options)
			if err != nil {
				errors <- err
			}
		}(devPod)
	}

	aggregatedErrors := []error{}
	for _, initChan := range initChans {
		select {
		case err := <-errors:
			cancel()
			aggregatedErrors = append(aggregatedErrors, err)
			<-initChan
		case <-initChan:
		}
	}

	return utilerrors.NewAggregate(aggregatedErrors)
}

type DevPodAlreadyExists struct{}

func (DevPodAlreadyExists) Error() string {
	return "dev pod already exists, please make sure to stop the dev pod before rerunning it"
}

func (d *devPodManager) Wait() error {
	devPods := map[string]*devPod{}
	d.m.Lock()
	for k, v := range d.devPods {
		devPods[k] = v
	}
	d.m.Unlock()

	errors := []error{}
	for _, dp := range devPods {
		<-dp.Done()

		err := dp.Err()
		if err != nil {
			errors = append(errors, err)
		}
	}

	return utilerrors.NewAggregate(errors)
}

func (d *devPodManager) Start(originalContext devspacecontext.Context, devPodConfig *latest.DevPod, options Options) (*devPod, error) {
	lock := d.lockFactory.GetLock(devPodConfig.Name)
	lock.Lock()
	defer lock.Unlock()

	var dp *devPod
	d.m.Lock()
	dp = d.devPods[devPodConfig.Name]
	if dp != nil {
		select {
		case <-dp.Done():
		default:
			d.m.Unlock()
			return nil, DevPodAlreadyExists{}
		}
	}

	// create a new dev pod
	dp = newDevPod()
	d.devPods[devPodConfig.Name] = dp
	d.m.Unlock()

	// create a DevPod logger
	prefix := "dev:" + devPodConfig.Name + " "
	unionLogger := originalContext.Log().WithPrefix(prefix).WithSink(logpkg.GetDevPodFileLogger(prefix))

	// start the dev pod
	err := dp.Start(originalContext.WithLogger(unionLogger), devPodConfig, options)
	if err != nil {
		return nil, err
	}

	return dp, nil
}

func (d *devPodManager) Reset(ctx devspacecontext.Context, name string, options *deploy.PurgeOptions) error {
	lock := d.lockFactory.GetLock(name)
	lock.Lock()
	defer lock.Unlock()

	d.stop(name)
	devPod, ok := ctx.Config().RemoteCache().GetDevPod(name)
	if ok {
		_, err := podreplace.NewPodReplacer().RevertReplacePod(ctx, &devPod, options)
		return err
	}

	return nil
}

func (d *devPodManager) Stop(ctx devspacecontext.Context, name string) {
	lock := d.lockFactory.GetLock(name)
	lock.Lock()
	defer lock.Unlock()

	d.stop(name)
}

func (d *devPodManager) stop(name string) {
	d.m.Lock()
	dp := d.devPods[name]
	d.m.Unlock()
	if dp == nil {
		return
	}

	// stop the dev pod
	dp.Stop()
	d.m.Lock()
	delete(d.devPods, name)
	d.m.Unlock()
}
