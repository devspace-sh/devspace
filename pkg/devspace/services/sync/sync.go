package sync

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/services/runner"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/sirupsen/logrus"
)

// StartSyncFromCmd starts a new sync from command
func StartSyncFromCmd(ctx *devspacecontext.Context, selector targetselector.TargetSelector, syncConfig *latest.SyncConfig, noWatch, verbose bool) error {
	syncDone := make(chan struct{})
	options := &Options{
		SyncConfig:     syncConfig,
		Selector:       selector,
		RestartOnError: true,

		Done:    syncDone,
		SyncLog: ctx.Log,

		Verbose: verbose,
	}

	cancelCtx, cancel := context.WithCancel(ctx.Context)
	defer cancel()
	ctx = ctx.WithContext(cancelCtx)
	err := NewController().Start(ctx, options)
	if err != nil {
		return err
	}

	// Handle no watch
	if noWatch {
		cancel()
		<-syncDone
		return nil
	}

	// Handle interrupt
	select {
	case <-ctx.Context.Done():
		return nil
	case <-syncDone:
		return nil
	}
}

// StartSync starts the syncing functionality
func StartSync(ctx *devspacecontext.Context, devPod *latest.DevPod, selector targetselector.TargetSelector, done chan struct{}) error {
	if ctx == nil || ctx.Config == nil || ctx.Config.Config() == nil {
		return fmt.Errorf("DevSpace config is nil")
	}

	// Start sync client
	doneChans := []chan struct{}{}
	r := runner.NewRunner(5)
	for _, syncConfig := range devPod.Sync {
		doneChan := make(chan struct{})
		doneChans = append(doneChans, doneChan)
		err := r.Run(newSyncFn(ctx, devPod.Name, string(devPod.Arch), syncConfig, selector.WithContainer(devPod.Container), doneChan))
		if err != nil {
			if done != nil {
				close(done)
			}
			return err
		}
	}
	for _, c := range devPod.Containers {
		for _, syncConfig := range c.Sync {
			doneChan := make(chan struct{})
			doneChans = append(doneChans, doneChan)
			err := r.Run(newSyncFn(ctx, devPod.Name, string(c.Arch), syncConfig, selector.WithContainer(c.Container), doneChan))
			if err != nil {
				if done != nil {
					close(done)
				}
				return err
			}
		}
	}

	if done != nil {
		go func() {
			for i := 0; i < len(doneChans); i++ {
				<-doneChans[i]
			}

			close(done)
		}()
	}

	return r.Wait()
}

func newSyncFn(ctx *devspacecontext.Context, name, arch string, syncConfig *latest.SyncConfig, selector targetselector.TargetSelector, done chan struct{}) func() error {
	return func() error {
		// set options
		options := &Options{
			Name:       name,
			Selector:   selector,
			SyncConfig: syncConfig,
			Arch:       arch,

			RestartOnError: true,
			Done:           done,
			Verbose:        ctx.Log.GetLevel() == logrus.DebugLevel,
		}

		// should we print the logs?
		if syncConfig.PrintLogs {
			options.SyncLog = ctx.Log
		} else {
			options.SyncLog = logpkg.GetDevPodFileLogger(name)
		}

		return NewController().Start(ctx, options)
	}
}
