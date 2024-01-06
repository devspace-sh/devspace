package sync

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/devspace/sync"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/sirupsen/logrus"
)

// StartSyncFromCmd starts a new sync from command
func StartSyncFromCmd(ctx devspacecontext.Context, selector targetselector.TargetSelector, name string, syncConfig *latest.SyncConfig, noWatch bool) error {
	ctx, parent := ctx.WithNewTomb()
	options := &Options{
		Name:           name,
		SyncConfig:     syncConfig,
		Selector:       selector,
		RestartOnError: true,
		SyncLog:        ctx.Log(),

		Verbose: ctx.Log().GetLevel() == logrus.DebugLevel,
	}

	// Start the tomb
	<-parent.NotifyGo(func() error {
		// this is needed as otherwise the context
		// is cancelled alongside the tomb
		parent.Go(func() error {
			<-ctx.Context().Done()
			return nil
		})

		return NewController().Start(ctx, options, parent)
	})

	// Handle no watch
	if noWatch {
		select {
		case <-parent.Dead():
			return parent.Err()
		default:
			parent.Kill(nil)
			_ = parent.Wait()
			return nil
		}
	}

	// Handle interrupt
	select {
	case <-parent.Dead():
		return parent.Err()
	case <-ctx.Context().Done():
		_ = parent.Wait()
		return nil
	}
}

// StartSync starts the syncing functionality
func StartSync(ctx devspacecontext.Context, devPod *latest.DevPod, selector targetselector.TargetSelector, parent *tomb.Tomb) (retErr error) {
	if ctx == nil || ctx.Config() == nil || ctx.Config().Config() == nil {
		return fmt.Errorf("DevSpace config is nil")
	}

	// init done array is used to track when sync was initialized
	initDoneArray := []chan struct{}{}
	loader.EachDevContainer(devPod, func(devContainer *latest.DevContainer) bool {
		starter := sync.NewDelayedContainerStarter()

		// make sure we add all the sync paths that need to wait for initial start
		for _, syncConfig := range devContainer.Sync {
			if syncConfig.StartContainer || (syncConfig.OnUpload != nil && syncConfig.OnUpload.RestartContainer) {
				starter.Inc()
			}
		}

		// now start the sync paths
		for _, syncConfig := range devContainer.Sync {
			// start a new go routine in the tomb
			s := syncConfig
			syncCtx := ctx
			var cancel context.CancelFunc
			if s.NoWatch {
				var cancelCtx context.Context
				cancelCtx, cancel = context.WithCancel(syncCtx.Context())
				syncCtx = syncCtx.WithContext(cancelCtx)
			}
			initDone := parent.NotifyGo(func() error {
				if cancel != nil {
					defer cancel()
				}

				return startSync(syncCtx, devPod.Name, string(devContainer.Arch), s, selector.WithContainer(devContainer.Container), starter, parent)
			})
			initDoneArray = append(initDoneArray, initDone)

			// every five we wait
			if len(initDoneArray)%5 == 0 {
				for _, initDone := range initDoneArray {
					<-initDone
				}
			}
		}
		return true
	})

	// wait for init chans to be finished
	for _, initDone := range initDoneArray {
		<-initDone
	}
	return nil
}

func startSync(ctx devspacecontext.Context, name, arch string, syncConfig *latest.SyncConfig, selector targetselector.TargetSelector, starter sync.DelayedContainerStarter, parent *tomb.Tomb) error {
	// set options
	options := &Options{
		Name:       name,
		Selector:   selector,
		SyncConfig: syncConfig,
		Arch:       arch,
		Starter:    starter,

		RestartOnError: true,
		Verbose:        ctx.Log().GetLevel() == logrus.DebugLevel,
	}

	// should we print the logs?
	if syncConfig.PrintLogs || ctx.Log().GetLevel() == logrus.DebugLevel {
		options.SyncLog = ctx.Log()
	} else {
		options.SyncLog = logpkg.GetDevPodFileLogger(name)
	}

	return NewController().Start(ctx, options, parent)
}
