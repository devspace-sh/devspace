package devpod

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/services/podreplace"
	"github.com/loft-sh/devspace/pkg/util/lockfactory"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/stringutil"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sync"
	"time"
)

type Manager interface {
	// StartMultiple will start multiple or all dev pods
	StartMultiple(ctx *devspacecontext.Context, devPods []string) error

	// Reset will stop the DevPod if it exists and reset the replaced pods
	Reset(ctx *devspacecontext.Context, name string) error

	// List lists the currently active dev pods
	List() []string

	// Stop will stop the DevPod
	Stop(name string)

	// Close will close the manager and wait for all dev pods
	Close()

	// Context returns the context of the DevManager
	Context() context.Context

	// Wait will wait until all DevPods are stopped
	Wait()
}

type devPodManager struct {
	lockFactory lockfactory.LockFactory

	m       sync.Mutex
	ctx     context.Context
	cancels []context.CancelFunc
	devPods map[string]*devPod
}

func NewManager(ctx context.Context) Manager {
	ctx, cancel := context.WithCancel(ctx)
	return &devPodManager{
		ctx:         ctx,
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
	d.Wait()
}

func (d *devPodManager) Context() context.Context {
	return d.ctx
}

func (d *devPodManager) StartMultiple(ctx *devspacecontext.Context, devPods []string) error {
	select {
	case <-d.ctx.Done():
		return d.ctx.Err()
	default:
	}

	cancelCtx, cancel := context.WithCancel(d.ctx)
	d.m.Lock()
	d.cancels = append(d.cancels, cancel)
	d.m.Unlock()
	ctx = ctx.WithContext(cancelCtx)

	initChans := []chan struct{}{}
	errors := make(chan error, len(ctx.Config.Config().Dev))
	for devPodName, devPod := range ctx.Config.Config().Dev {
		if len(devPods) > 0 && !stringutil.Contains(devPods, devPodName) {
			continue
		}

		initChan := make(chan struct{})
		initChans = append(initChans, initChan)
		go func(devPod *latest.DevPod) {
			defer close(initChan)

			_, err := d.Start(ctx, devPod)
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

func (d *devPodManager) Wait() {
	devPods := map[string]*devPod{}
	d.m.Lock()
	for k, v := range d.devPods {
		devPods[k] = v
	}
	d.m.Unlock()

	for _, dp := range devPods {
		<-dp.Done()
	}
}

func (d *devPodManager) Start(originalContext *devspacecontext.Context, devPodConfig *latest.DevPod) (*devPod, error) {
	lock := d.lockFactory.GetLock(devPodConfig.Name)
	lock.Lock()
	defer lock.Unlock()

	var dp *devPod
	d.m.Lock()
	dp = d.devPods[devPodConfig.Name]
	d.m.Unlock()

	// check if already running
	if dp != nil && dp.Alive() {
		return nil, DevPodAlreadyExists{}
	}

	// create a new dev pod
	dp = newDevPod()
	d.m.Lock()
	d.devPods[devPodConfig.Name] = dp
	d.m.Unlock()

	// create a DevPod logger
	prefix := devPodConfig.Name + " "
	unionLogger := logpkg.NewUnionLogger(originalContext.Log.GetLevel(), logpkg.NewDefaultPrefixLogger(prefix, originalContext.Log.WithoutPrefix()), logpkg.GetDevPodFileLogger(prefix))

	// start the dev pod
	err := dp.Start(originalContext.WithLogger(unionLogger), devPodConfig)
	if err != nil {
		return nil, err
	}

	// restart dev pod if necessary
	go func() {
		<-dp.Done()
		if originalContext.IsDone() {
			return
		}

		// try restarting the dev pod if it has stopped because of
		// a lost connection
		if _, ok := dp.Err().(DevPodLostConnection); ok {
			for {
				_, err = d.Start(originalContext, devPodConfig)
				if err != nil {
					if originalContext.IsDone() {
						return
					} else if _, ok := err.(DevPodAlreadyExists); ok {
						return
					}

					originalContext.Log.Infof("Restart dev %s because of: %v", devPodConfig.Name, err)
					time.Sleep(time.Second * 10)
					continue
				}

				return
			}
		}
	}()

	return dp, nil
}

func (d *devPodManager) Reset(ctx *devspacecontext.Context, name string) error {
	lock := d.lockFactory.GetLock(name)
	lock.Lock()
	defer lock.Unlock()

	d.m.Lock()
	defer d.m.Unlock()

	d.stop(name)
	devPod, ok := ctx.Config.RemoteCache().GetDevPod(name)
	if ok {
		_, err := podreplace.NewPodReplacer().RevertReplacePod(ctx, &devPod)
		return err
	}

	return nil
}

func (d *devPodManager) Stop(name string) {
	lock := d.lockFactory.GetLock(name)
	lock.Lock()
	defer lock.Unlock()

	d.m.Lock()
	defer d.m.Unlock()

	d.stop(name)
}

func (d *devPodManager) stop(name string) {
	dp := d.devPods[name]
	if dp == nil {
		return
	}

	// stop the dev pod
	dp.Stop()
	delete(d.devPods, name)
}
