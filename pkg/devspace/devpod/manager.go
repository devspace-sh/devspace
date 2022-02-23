package devpod

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/util/lockfactory"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/stringutil"
	"sync"
)

type Manager interface {
	// StartMultiple will start multiple or all dev pods
	StartMultiple(ctx *devspacecontext.Context, devPods []string) error

	// Start will start a DevPod if it is not yet started
	Start(ctx *devspacecontext.Context, devPod *latest.DevPod) error

	// Stop will stop the DevPod
	Stop(ctx *devspacecontext.Context, name string, log logpkg.Logger) error
}

type devPodManager struct {
	lockFactory lockfactory.LockFactory

	mapLock sync.Mutex
	devPods map[string]*devPod
}

func NewManager() Manager {
	return &devPodManager{
		lockFactory: lockfactory.NewDefaultLockFactory(),
		devPods:     map[string]*devPod{},
	}
}

func (d *devPodManager) StartMultiple(ctx *devspacecontext.Context, devPods []string) error {
	cancelCtx, cancel := context.WithCancel(ctx.Context)
	defer cancel()
	ctx = ctx.WithContext(cancelCtx)

	waitGroup := sync.WaitGroup{}
	errChan := make(chan error, len(ctx.Config.Config().Dev))
	for devPodName, devPod := range ctx.Config.Config().Dev {
		if len(devPods) > 0 && !stringutil.Contains(devPods, devPodName) {
			continue
		}

		waitGroup.Add(1)
		go func(devPod *latest.DevPod) {
			defer waitGroup.Done()
			err := d.Start(ctx, devPod)
			if err != nil {
				errChan <- err
			}
		}(devPod)
	}

	done := make(chan struct{})
	go func() {
		waitGroup.Wait()
		close(done)
	}()

	select {
	case err := <-errChan:
		cancel()
		<-done
		return err
	case <-done:
		return nil
	}
}

func (d *devPodManager) Start(ctx *devspacecontext.Context, devPodConfig *latest.DevPod) error {
	lock := d.lockFactory.GetLock(devPodConfig.Name)
	lock.Lock()
	defer lock.Unlock()

	// stop the dev pod
	var dp *devPod
	d.mapLock.Lock()
	dp = d.devPods[devPodConfig.Name]
	d.mapLock.Unlock()
	if dp != nil {
		return fmt.Errorf("dev pod already exists, please make sure to stop the dev pod before rerunning it")
	}

	// create a DevPod logger
	prefix := devPodConfig.Name
	unionLogger := logpkg.NewUnionLogger(logpkg.NewDefaultPrefixLogger(prefix, ctx.Log), logpkg.GetDevPodFileLogger(prefix))
	ctx = ctx.WithLogger(unionLogger)

	// start the dev pod
	dp = newDevPod()
	err := dp.Start(ctx, devPodConfig)
	if err != nil {
		return err
	}

	// save dev pod in the map
	d.mapLock.Lock()
	defer d.mapLock.Unlock()

	d.devPods[devPodConfig.Name] = dp
	return nil
}

func (d *devPodManager) Stop(ctx *devspacecontext.Context, name string, log logpkg.Logger) error {
	lock := d.lockFactory.GetLock(name)
	lock.Lock()
	defer lock.Unlock()

	// stop the dev pod
	var dp *devPod
	d.mapLock.Lock()
	dp = d.devPods[name]
	d.mapLock.Unlock()
	if dp == nil {
		return nil
	}

	// stop the dev pod
	log.Infof("Stop dev pod %s", name)
	dp.Stop()

	// now remove from map
	d.mapLock.Lock()
	defer d.mapLock.Unlock()

	delete(d.devPods, name)
	return nil
}
