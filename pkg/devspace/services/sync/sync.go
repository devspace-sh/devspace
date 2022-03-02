package sync

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/sirupsen/logrus"
)

// StartSyncFromCmd starts a new sync from command
func StartSyncFromCmd(ctx *devspacecontext.Context, selector targetselector.TargetSelector, syncConfig *latest.SyncConfig, noWatch, verbose bool) error {
	ctx, parent := ctx.WithNewTomb()
	options := &Options{
		SyncConfig:     syncConfig,
		Selector:       selector,
		RestartOnError: true,
		SyncLog:        ctx.Log,

		Verbose: verbose,
	}

	// Start the tomb
	<-parent.NotifyGo(func() error {
		return NewController().Start(ctx, options, parent)
	})

	// Handle no watch
	if noWatch {
		parent.Kill(nil)
		_ = parent.Wait()
		return nil
	}

	// Handle interrupt
	select {
	case <-ctx.Context.Done():
		_ = parent.Wait()
		return nil
	case <-parent.Dead():
		return parent.Err()
	}
}

// StartSync starts the syncing functionality
func StartSync(ctx *devspacecontext.Context, devPod *latest.DevPod, selector targetselector.TargetSelector, parent *tomb.Tomb) (retErr error) {
	if ctx == nil || ctx.Config == nil || ctx.Config.Config() == nil {
		return fmt.Errorf("DevSpace config is nil")
	}

	// init done array is used to track when sync was initialized
	initDoneArray := []chan struct{}{}
	loader.EachDevContainer(devPod, func(devContainer *latest.DevContainer) bool {
		for i, syncConfig := range devContainer.Sync {
			// start a new go routine in the tomb
			initDoneArray = append(initDoneArray, parent.NotifyGo(func() error {
				return startSync(ctx, devPod.Name, string(devContainer.Arch), syncConfig, selector.WithContainer(devContainer.Container), parent)
			}))

			// every five we wait
			if i%5 == 0 {
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

func startSync(ctx *devspacecontext.Context, name, arch string, syncConfig *latest.SyncConfig, selector targetselector.TargetSelector, parent *tomb.Tomb) error {
	// set options
	options := &Options{
		Name:       name,
		Selector:   selector,
		SyncConfig: syncConfig,
		Arch:       arch,

		RestartOnError: true,
		Verbose:        ctx.Log.GetLevel() == logrus.DebugLevel,
	}

	// should we print the logs?
	if syncConfig.PrintLogs || ctx.Log.GetLevel() == logrus.DebugLevel {
		options.SyncLog = ctx.Log
	} else {
		options.SyncLog = logpkg.GetDevPodFileLogger(name)
	}

	return NewController().Start(ctx, options, parent)
}
