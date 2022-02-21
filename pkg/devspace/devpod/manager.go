package devpod

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/util/lockfactory"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"sync"
)

var GlobalManager = NewManager()

type Manager interface {
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
