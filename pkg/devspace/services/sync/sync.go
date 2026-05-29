package sync

import (
	"context"
	"fmt"
	stdsync "sync"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/services/sync/synctarget"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/devspace/sync"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/sirupsen/logrus"
)

// StartSyncFromCmd starts a new sync from command
func StartSyncFromCmd(ctx devspacecontext.Context, selector targetselector.TargetSelector, name string, syncConfig *latest.SyncConfig, noWatch bool) error {
	ctx, parent := ctx.WithNewTomb()
	<-parent.NotifyGo(func() error {
		parent.Go(func() error {
			<-ctx.Context().Done()
			return nil
		})
		return startSync(ctx, name, "", syncConfig, selector, nil, parent)
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
			if syncConfig.SyncReplicas {
				continue
			}
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
	targets, err := synctarget.BuildTargets(ctx.Context(), ctx.Log(), ctx.KubeClient(), selector, syncConfig)
	if err != nil {
		return err
	}

	// One tomb per replica: the sync controller calls Kill on it and would stop sibling syncs.
	if len(targets) == 1 {
		return startSyncOneTarget(ctx, name, arch, syncConfig, starter, parent, targets, 0)
	}

	var wg stdsync.WaitGroup
	errs := make([]error, len(targets))
	for i := range targets {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			var subTomb tomb.Tomb
			errs[i] = startSyncOneTarget(ctx, name, arch, syncConfig, starter, &subTomb, targets, i)
		}()
	}
	wg.Wait()
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

func startSyncOneTarget(
	ctx devspacecontext.Context,
	name, arch string,
	syncConfig *latest.SyncConfig,
	starter sync.DelayedContainerStarter,
	syncParent *tomb.Tomb,
	targets []synctarget.Target,
	i int,
) error {
	target := targets[i]
	syncConfigForTarget := synctarget.ConfigForIndex(syncConfig, i)
	if synctarget.ReplicasEnabled(syncConfig) && target.Pod != "" && target.Container != "" {
		ctx.Log().Infof(
			"Sync target %d/%d: %s/%s:%s (disableUpload=%t disableDownload=%t)",
			i+1,
			len(targets),
			target.Namespace,
			target.Pod,
			target.Container,
			syncConfigForTarget.DisableUpload,
			syncConfigForTarget.DisableDownload,
		)
	}

	effectiveStarter := starter
	if synctarget.ReplicasEnabled(syncConfig) && (syncConfig.StartContainer || (syncConfig.OnUpload != nil && syncConfig.OnUpload.RestartContainer)) {
		ts := sync.NewDelayedContainerStarter()
		ts.Inc()
		effectiveStarter = ts
	}

	options := &Options{
		Name:       name,
		Selector:   target.Selector,
		SyncConfig: syncConfigForTarget,
		Arch:       arch,
		Starter:    effectiveStarter,

		RestartOnError: true,
		Verbose:        ctx.Log().GetLevel() == logrus.DebugLevel,
	}

	if syncConfigForTarget.PrintLogs || ctx.Log().GetLevel() == logrus.DebugLevel {
		options.SyncLog = ctx.Log()
	} else {
		options.SyncLog = logpkg.GetDevPodFileLogger(name)
	}

	return NewController().Start(ctx, options, syncParent)
}
