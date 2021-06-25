package services

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/services/synccontroller"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sync"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"

	"github.com/pkg/errors"
)

// StartSyncFromCmd starts a new sync from command
func (serviceClient *client) StartSyncFromCmd(targetOptions targetselector.Options, syncConfig *latest.SyncConfig, interrupt chan error, verbose bool) error {
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

	if syncConfig.WaitInitialSync == nil || *syncConfig.WaitInitialSync == true {
		return nil
	}

	// Wait till sync is finished
	<-syncDone
	return nil
}

// StartSync starts the syncing functionality
func (serviceClient *client) StartSync(interrupt chan error, printSyncLog bool, verboseSync bool) error {
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
		prefix := fmt.Sprintf("[%d:sync] ", idx)
		if syncConfig.ImageName != "" {
			prefix = fmt.Sprintf("[%d:sync:%s] ", idx, syncConfig.ImageName)
		}

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
		go func() {
			defer waitGroup.Done()

			err := synccontroller.NewController(serviceClient.config, serviceClient.dependencies, serviceClient.client, serviceClient.log).Start(options, log)
			if err != nil {
				errsMutex.Lock()
				errs = append(errs, errors.Errorf("unable to start sync: %v", err))
				errsMutex.Unlock()
			}
		}()

		// every 5 we wait
		if idx%5 == 4 {
			waitGroup.Wait()
		}
	}

	waitGroup.Wait()
	return utilerrors.NewAggregate(errs)
}
