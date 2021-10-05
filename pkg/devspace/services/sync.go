package services

import (
	"fmt"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/services/synccontroller"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
)

// StartSyncFromCmd starts a new sync from command
func (serviceClient *client) StartSyncFromCmd(targetOptions targetselector.Options, syncConfig *latest.SyncConfig, interrupt chan error, noWatch, verbose bool) error {
	syncDone := make(chan struct{})
	options := &synccontroller.Options{
		Interrupt: interrupt,

		SyncConfig:    syncConfig,
		TargetOptions: targetOptions,

		RestartOnError: true,

		Done:    syncDone,
		SyncLog: serviceClient.log,

		Verbose: verbose,
	}

	err := synccontroller.NewController(serviceClient.config, serviceClient.dependencies, serviceClient.client).Start(options, serviceClient.log)
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

type PrefixFn func(idx int, name, operation string) string

func DependencyPrefixFn(dependency string) PrefixFn {
	return func(idx int, name, operation string) string {
		prefix := fmt.Sprintf("[%s:%d:%s] ", dependency, idx, operation)
		if name != "" {
			prefix = fmt.Sprintf("[%s:%s] ", dependency, name)
		}

		return prefix
	}
}

func DefaultPrefixFn(idx int, name, operation string) string {
	prefix := fmt.Sprintf("[%d:%s] ", idx, operation)
	if name != "" {
		prefix = fmt.Sprintf("[%s] ", name)
	}

	return prefix
}

// StartSync starts the syncing functionality
func (serviceClient *client) StartSync(interrupt chan error, printSyncLog, verboseSync bool, prefixFn PrefixFn) error {
	if serviceClient.config == nil || serviceClient.config.Config() == nil {
		return fmt.Errorf("DevSpace config is nil")
	}

	// Start sync client
	runner := NewRunner(5)
	for idx, syncConfig := range serviceClient.config.Config().Dev.Sync {
		err := runner.Run(serviceClient.newSyncFn(idx, syncConfig, interrupt, printSyncLog, verboseSync, prefixFn))
		if err != nil {
			return err
		}
	}

	return runner.Wait()
}

func (serviceClient *client) newSyncFn(idx int, syncConfig *latest.SyncConfig, interrupt chan error, printSyncLog, verboseSync bool, prefixFn PrefixFn) func() error {
	return func() error {
		targetOptions := targetselector.NewEmptyOptions().ApplyConfigParameter(syncConfig.LabelSelector, syncConfig.Namespace, syncConfig.ContainerName, "")
		targetOptions.AllowPick = false
		targetOptions.WaitingStrategy = targetselector.NewUntilNewestRunningWaitingStrategy(time.Second * 2)

		// set options
		options := &synccontroller.Options{
			Interrupt: interrupt,

			SyncConfig:    syncConfig,
			TargetOptions: targetOptions,

			RestartOnError: true,
			Verbose:        verboseSync,
		}

		// should we print the logs?
		prefix := prefixFn(idx, syncConfig.Name, "sync")
		fileLog := logpkg.NewPrefixLogger(prefix, "", logpkg.GetFileLogger("sync"))
		log := logpkg.NewDefaultPrefixLogger(prefix, serviceClient.log)
		if printSyncLog {
			unionLog := logpkg.NewUnionLogger(log, fileLog)
			options.SyncLog = unionLog
		} else {
			options.SyncLog = fileLog
		}

		return synccontroller.NewController(serviceClient.config, serviceClient.dependencies, serviceClient.client).Start(options, log)
	}
}
