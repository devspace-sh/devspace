package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/services/synccontroller"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"

	"github.com/pkg/errors"
)

// StartSyncFromCmd starts a new sync from command
func (serviceClient *client) StartSyncFromCmd(targetOptions targetselector.Options, syncConfig *latest.SyncConfig, interrupt chan error, noWatch, verbose bool) error {
	syncDone := make(chan struct{})
	options := &synccontroller.Options{
		Interrupt: interrupt,

		SyncConfig:    syncConfig,
		TargetOptions: targetOptions,

		RestartOnError: true,
		RestartLog:     serviceClient.log,

		Done:    syncDone,
		SyncLog: serviceClient.log,

		Verbose: verbose,
	}

	err := synccontroller.NewController(serviceClient.config, serviceClient.dependencies, serviceClient.client, serviceClient.log).Start(options, serviceClient.log)
	if err != nil {
		return err
	}

	// Handle no watch
	if noWatch {
		return nil
	}

	// Handle interrupt
	if options.Interrupt != nil {
		for {
			select {
			case <-syncDone:
				return nil
			case <-options.Interrupt:
				return nil
			}
		}
	}

	// Wait till sync is finished
	<-syncDone
	return nil
}

func DefaultPrefixFn(idx int, syncConfig *latest.SyncConfig) string {
	prefix := fmt.Sprintf("[%d:sync] ", idx)
	if syncConfig.Name != "" {
		prefix = fmt.Sprintf("[%s] ", syncConfig.Name)
	}

	return prefix
}

// StartSync starts the syncing functionality
func (serviceClient *client) StartSync(interrupt chan error, printSyncLog, verboseSync bool, prefixFn func(idx int, syncConfig *latest.SyncConfig) string) error {
	if serviceClient.config == nil || serviceClient.config.Config() == nil {
		return fmt.Errorf("DevSpace config is nil")
	}

	// Start sync client
	waitGroup := sync.WaitGroup{}
	errs := []error{}
	errsMutex := sync.Mutex{}
	for idx, syncConfig := range serviceClient.config.Config().Dev.Sync {
		targetOptions := targetselector.NewEmptyOptions().ApplyConfigParameter(syncConfig.LabelSelector, syncConfig.Namespace, syncConfig.ContainerName, "")
		targetOptions.AllowPick = false
		targetOptions.WaitingStrategy = targetselector.NewUntilNewestRunningWaitingStrategy(time.Second * 2)

		// set options
		options := &synccontroller.Options{
			Interrupt: interrupt,

			SyncConfig:    syncConfig,
			TargetOptions: targetOptions,

			RestartOnError: true,
			RestartLog:     logpkg.Discard,
			Verbose:        verboseSync,
		}

		// should we print the logs?
		prefix := prefixFn(idx, syncConfig)
		log := logpkg.NewPrefixLogger(prefix, logpkg.Colors[idx%len(logpkg.Colors)], serviceClient.log)
		if printSyncLog {
			options.SyncLog = log
			options.RestartLog = log
		} else {
			fileLog := logpkg.NewPrefixLogger(prefix, "", logpkg.GetFileLogger("sync"))
			options.SyncLog = fileLog
			options.RestartLog = fileLog
		}

		waitGroup.Add(1)
		go func(options *synccontroller.Options) {
			defer waitGroup.Done()

			err := synccontroller.NewController(serviceClient.config, serviceClient.dependencies, serviceClient.client, serviceClient.log).Start(options, log)
			if err != nil {
				errsMutex.Lock()
				errs = append(errs, errors.Errorf("unable to start sync: %v", err))
				errsMutex.Unlock()
			}
		}(options)

		// every 5 we wait
		if idx%5 == 4 {
			waitGroup.Wait()
		}
	}

	waitGroup.Wait()
	return utilerrors.NewAggregate(errs)
}
