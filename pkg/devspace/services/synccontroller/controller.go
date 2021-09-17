package synccontroller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/util"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/inject"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/devspace/sync"
	"github.com/loft-sh/devspace/pkg/util/imageselector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/scanner"
	"github.com/moby/buildkit/frontend/dockerfile/dockerignore"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

type Controller interface {
	Start(options *Options, log logpkg.Logger) error
}

func NewController(config config.Config, dependencies []types.Dependency, client kubectl.Client, log logpkg.Logger) Controller {
	return &controller{
		config:       config,
		dependencies: dependencies,
		client:       client,
		log:          log,
	}
}

type controller struct {
	config       config.Config
	dependencies []types.Dependency
	client       kubectl.Client
	log          logpkg.Logger
}

type Options struct {
	SyncConfig    *latest.SyncConfig
	TargetOptions targetselector.Options

	Interrupt chan error
	Done      chan struct{}

	RestartOnError bool
	RestartLog     logpkg.Logger

	SyncLog logpkg.Logger
	Verbose bool
}

func (c *controller) Start(options *Options, log logpkg.Logger) error {
	pluginErr := hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
		"sync_config": options.SyncConfig,
	}, log, hook.EventsForSingle("start:sync", options.SyncConfig.Name).With("sync.start")...)
	if pluginErr != nil {
		return pluginErr
	}

	err := c.startWithWait(options, log)
	if err != nil {
		pluginErr := hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
			"sync_config": options.SyncConfig,
			"ERROR":       err,
		}, log, hook.EventsForSingle("error:sync", options.SyncConfig.Name).With("sync.error")...)
		if pluginErr != nil {
			return pluginErr
		}

		return err
	}

	return nil
}

func (c *controller) startWithWait(options *Options, log logpkg.Logger) error {
	var (
		onInitUploadDone   chan struct{}
		onInitDownloadDone chan struct{}
		onError            = make(chan error)
		onDone             = make(chan struct{})
	)

	// should wait for initial sync?
	if options.SyncConfig.WaitInitialSync == nil || *options.SyncConfig.WaitInitialSync {
		onInitUploadDone = make(chan struct{})
		onInitDownloadDone = make(chan struct{})
		pluginErr := hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
			"sync_config": options.SyncConfig,
		}, log, hook.EventsForSingle("before:initialSync", options.SyncConfig.Name).With("sync.beforeInitialSync")...)
		if pluginErr != nil {
			return pluginErr
		}
	}

	// start the sync
	client, err := c.startSync(options, onInitUploadDone, onInitDownloadDone, onDone, onError, log)
	if err != nil {
		pluginErr := hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
			"sync_config": options.SyncConfig,
			"ERROR":       err,
		}, log, hook.EventsForSingle("error:initialSync", options.SyncConfig.Name).With("sync.errorInitialSync")...)
		if pluginErr != nil {
			return pluginErr
		}

		return err
	}

	// should wait for initial sync?
	if options.SyncConfig.WaitInitialSync == nil || *options.SyncConfig.WaitInitialSync {
		log.Info("Waiting for initial sync to complete")
		var (
			uploadDone   = false
			downloadDone = false
		)
		for {
			select {
			case err := <-onError:
				pluginErr := hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
					"sync_config": options.SyncConfig,
					"ERROR":       err,
				}, log, hook.EventsForSingle("error:initialSync", options.SyncConfig.Name).With("sync.errorInitialSync")...)
				if pluginErr != nil {
					return pluginErr
				}
				return errors.Wrap(err, "initial sync")
			case <-onInitUploadDone:
				uploadDone = true
			case <-onInitDownloadDone:
				downloadDone = true
			case <-options.Interrupt:
				client.Stop(nil)
				pluginErr := hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
					"sync_config": options.SyncConfig,
				}, log, hook.EventsForSingle("stop:sync", options.SyncConfig.Name).With("sync.stop")...)
				if pluginErr != nil {
					return pluginErr
				}
				return nil
			case <-onDone:
				if options.Done != nil {
					close(options.Done)
				}
				pluginErr := hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
					"sync_config": options.SyncConfig,
				}, log, hook.EventsForSingle("stop:sync", options.SyncConfig.Name).With("sync.stop")...)
				if pluginErr != nil {
					return pluginErr
				}
				return nil
			}
			if uploadDone && downloadDone {
				break
			}
		}
		pluginErr := hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
			"sync_config": options.SyncConfig,
		}, log, hook.EventsForSingle("after:initialSync", options.SyncConfig.Name).With("sync.afterInitialSync")...)
		if pluginErr != nil {
			return pluginErr
		}
	}

	// should we restart the client on error?
	if options.RestartOnError {
		go func(syncClient *sync.Sync, options *Options) {
			select {
			case err = <-onError:
				hook.LogExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
					"sync_config": options.SyncConfig,
					"ERROR":       err,
				}, c.log, hook.EventsForSingle("restart:sync", options.SyncConfig.Name).With("sync.restart")...)

				options.RestartLog.Info("Restarting sync...")
				for {
					err := c.startWithWait(options, options.RestartLog)
					if err != nil {

						hook.LogExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
							"sync_config": options.SyncConfig,
							"ERROR":       err,
						}, c.log, hook.EventsForSingle("restart:sync", options.SyncConfig.Name).With("sync.restart")...)
						c.log.Errorf("Error restarting sync: %v", err)
						c.log.Errorf("Will try again in 15 seconds")
						time.Sleep(time.Second * 15)
						continue
					}

					break
				}
			case <-options.Interrupt:
				syncClient.Stop(nil)
				hook.LogExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
					"sync_config": options.SyncConfig,
				}, c.log, hook.EventsForSingle("stop:sync", options.SyncConfig.Name).With("sync.stop")...)
			case <-onDone:
				if options.Done != nil {
					close(options.Done)
				}
				hook.LogExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
					"sync_config": options.SyncConfig,
				}, c.log, hook.EventsForSingle("stop:sync", options.SyncConfig.Name).With("sync.stop")...)
			}
		}(client, options)
	}

	return nil
}

func (c *controller) startSync(options *Options, onInitUploadDone chan struct{}, onInitDownloadDone chan struct{}, onDone chan struct{}, onError chan error, log logpkg.Logger) (*sync.Sync, error) {
	options.TargetOptions.SkipInitContainers = true
	var (
		syncConfig = options.SyncConfig
	)

	localPath := "."
	if syncConfig.LocalSubPath != "" {
		localPath = syncConfig.LocalSubPath
	}

	// check if local path exists
	_, err := os.Stat(localPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		err = os.MkdirAll(localPath, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	options.TargetOptions.ImageSelector = []imageselector.ImageSelector{}
	if syncConfig.ImageSelector != "" {
		imageSelector, err := util.ResolveImageAsImageSelector(syncConfig.ImageSelector, c.config, c.dependencies)
		if err != nil {
			return nil, err
		}

		options.TargetOptions.ImageSelector = append(options.TargetOptions.ImageSelector, *imageSelector)
	}

	log.Info("Waiting for pods...")
	container, err := targetselector.NewTargetSelector(c.client).SelectSingleContainer(context.TODO(), options.TargetOptions, c.log)
	if err != nil {
		return nil, errors.Errorf("Error selecting pod: %v", err)
	}

	log.Info("Starting sync...")
	syncClient, err := c.initClient(container.Pod, container.Container.Name, syncConfig, options.Verbose, options.SyncLog)
	if err != nil {
		return nil, errors.Wrap(err, "start sync")
	}

	err = syncClient.Start(onInitUploadDone, onInitDownloadDone, onDone, onError)
	if err != nil {
		return nil, errors.Errorf("Sync error: %v", err)
	}

	containerPath := "."
	if syncConfig.ContainerPath != "" {
		containerPath = syncConfig.ContainerPath
	}

	log.Donef("Sync started on %s <-> %s (Pod: %s/%s)", syncClient.LocalPath, containerPath, container.Pod.Namespace, container.Pod.Name)
	return syncClient, nil
}

func (c *controller) isFatalSyncError(err error) bool {
	return strings.Contains(err.Error(), "You are trying to sync the complete container root")

}

func (c *controller) initClient(pod *v1.Pod, container string, syncConfig *latest.SyncConfig, verbose bool, customLog logpkg.Logger) (*sync.Sync, error) {
	err := inject.InjectDevSpaceHelper(c.client, pod, container, string(syncConfig.Arch), customLog)
	if err != nil {
		return nil, err
	}

	localPath := "."
	if syncConfig.LocalSubPath != "" {
		localPath = syncConfig.LocalSubPath
	}

	containerPath := "."
	if syncConfig.ContainerPath != "" {
		containerPath = syncConfig.ContainerPath
	}

	upstreamDisabled := false
	if syncConfig.DisableUpload != nil {
		upstreamDisabled = *syncConfig.DisableUpload
	}

	downstreamDisabled := false
	if syncConfig.DisableDownload != nil {
		downstreamDisabled = *syncConfig.DisableDownload
	}

	compareBy := latest.InitialSyncCompareByMTime
	if syncConfig.InitialSyncCompareBy != "" {
		compareBy = syncConfig.InitialSyncCompareBy
	}

	options := sync.Options{
		Verbose:              verbose,
		InitialSyncCompareBy: compareBy,
		InitialSync:          syncConfig.InitialSync,
		UpstreamDisabled:     upstreamDisabled,
		DownstreamDisabled:   downstreamDisabled,
		Log:                  customLog,
		Polling:              syncConfig.Polling,
	}

	// Initialize log
	if options.Log == nil {
		options.Log = logpkg.GetFileLogger("sync")
	}

	// Add onDownload hooks
	if syncConfig.OnDownload != nil && syncConfig.OnDownload.ExecLocal != nil {
		fileCmd, fileArgs, dirCmd, dirArgs := getSyncCommands(syncConfig.OnDownload.ExecLocal)
		options.FileChangeCmd = fileCmd
		options.FileChangeArgs = fileArgs
		options.DirCreateCmd = dirCmd
		options.DirCreateArgs = dirArgs
	}

	if len(syncConfig.ExcludePaths) > 0 {
		options.ExcludePaths = syncConfig.ExcludePaths
	}

	if syncConfig.ExcludeFile != "" {
		paths, err := parseExcludeFile(filepath.Join(syncConfig.LocalSubPath, syncConfig.ExcludeFile))
		if err != nil {
			return nil, errors.Wrap(err, "parse exclude file")
		}
		options.ExcludePaths = append(options.ExcludePaths, paths...)
	}

	if len(syncConfig.DownloadExcludePaths) > 0 {
		options.DownloadExcludePaths = syncConfig.DownloadExcludePaths
	}

	if syncConfig.DownloadExcludeFile != "" {
		paths, err := parseExcludeFile(filepath.Join(syncConfig.LocalSubPath, syncConfig.DownloadExcludeFile))
		if err != nil {
			return nil, errors.Wrap(err, "parse download exclude file")
		}
		options.DownloadExcludePaths = append(options.DownloadExcludePaths, paths...)
	}

	if len(syncConfig.UploadExcludePaths) > 0 {
		options.UploadExcludePaths = syncConfig.UploadExcludePaths
	}

	if syncConfig.UploadExcludeFile != "" {
		paths, err := parseExcludeFile(filepath.Join(syncConfig.LocalSubPath, syncConfig.UploadExcludeFile))
		if err != nil {
			return nil, errors.Wrap(err, "parse upload exclude file")
		}
		options.UploadExcludePaths = append(options.UploadExcludePaths, paths...)
	}

	if syncConfig.BandwidthLimits != nil {
		if syncConfig.BandwidthLimits.Download != nil {
			options.DownstreamLimit = *syncConfig.BandwidthLimits.Download * 1024
		}

		if syncConfig.BandwidthLimits.Upload != nil {
			options.UpstreamLimit = *syncConfig.BandwidthLimits.Upload * 1024
		}
	}

	// check if we should restart the container on upload
	if syncConfig.OnUpload != nil && syncConfig.OnUpload.RestartContainer {
		options.RestartContainer = true
	}
	if syncConfig.OnUpload != nil && syncConfig.OnUpload.ExecRemote != nil && syncConfig.OnUpload.ExecRemote.OnBatch != nil && syncConfig.OnUpload.ExecRemote.OnBatch.Command != "" {
		options.UploadBatchCmd = syncConfig.OnUpload.ExecRemote.OnBatch.Command
		options.UploadBatchArgs = syncConfig.OnUpload.ExecRemote.OnBatch.Args
	}

	syncClient, err := sync.NewSync(localPath, options)
	if err != nil {
		return nil, errors.Wrap(err, "create sync")
	}

	// Start upstream
	upstreamArgs := []string{inject.DevSpaceHelperContainerPath, "sync", "upstream"}
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		upstreamArgs = append(upstreamArgs, "--override-permissions")
	}
	for _, exclude := range options.ExcludePaths {
		upstreamArgs = append(upstreamArgs, "--exclude", exclude)
	}
	for _, exclude := range options.DownloadExcludePaths {
		upstreamArgs = append(upstreamArgs, "--exclude", exclude)
	}
	if syncConfig.OnUpload != nil && syncConfig.OnUpload.ExecRemote != nil {
		onUpload := syncConfig.OnUpload.ExecRemote
		fileCmd, fileArgs, dirCmd, dirArgs := getSyncCommands(onUpload)
		if fileCmd != "" {
			upstreamArgs = append(upstreamArgs, "--filechangecmd", fileCmd)
			for _, arg := range fileArgs {
				upstreamArgs = append(upstreamArgs, "--filechangeargs", arg)
			}
		}
		if dirCmd != "" {
			upstreamArgs = append(upstreamArgs, "--dircreatecmd", dirCmd)
			for _, arg := range dirArgs {
				upstreamArgs = append(upstreamArgs, "--dircreateargs", arg)
			}
		}
	}

	upstreamArgs = append(upstreamArgs, containerPath)

	upStdinReader, upStdinWriter := io.Pipe()
	upStdoutReader, upStdoutWriter := io.Pipe()

	go func() {
		err := StartStream(c.client, pod, container, upstreamArgs, upStdinReader, upStdoutWriter, true, options.Log)
		if err != nil {
			syncClient.Stop(errors.Errorf("Sync - connection lost to pod %s/%s: %v", pod.Namespace, pod.Name, err))
		}
	}()

	err = syncClient.InitUpstream(upStdoutReader, upStdinWriter)
	if err != nil {
		return nil, errors.Wrap(err, "init upstream")
	}

	// Start downstream
	downstreamArgs := []string{inject.DevSpaceHelperContainerPath, "sync", "downstream"}
	if syncConfig.ThrottleChangeDetection != nil {
		downstreamArgs = append(downstreamArgs, "--throttle", strconv.FormatInt(*syncConfig.ThrottleChangeDetection, 10))
	}
	if syncConfig.Polling {
		downstreamArgs = append(downstreamArgs, "--polling")
	}
	for _, exclude := range options.ExcludePaths {
		downstreamArgs = append(downstreamArgs, "--exclude", exclude)
	}
	for _, exclude := range options.DownloadExcludePaths {
		downstreamArgs = append(downstreamArgs, "--exclude", exclude)
	}
	downstreamArgs = append(downstreamArgs, containerPath)

	downStdinReader, downStdinWriter := io.Pipe()
	downStdoutReader, downStdoutWriter := io.Pipe()

	go func() {
		err := StartStream(c.client, pod, container, downstreamArgs, downStdinReader, downStdoutWriter, true, options.Log)
		if err != nil {
			syncClient.Stop(errors.Errorf("Sync - connection lost to pod %s/%s: %v", pod.Namespace, pod.Name, err))
		}
	}()

	err = syncClient.InitDownstream(downStdoutReader, downStdinWriter)
	if err != nil {
		return nil, errors.Wrap(err, "init downstream")
	}

	return syncClient, nil
}

func getSyncCommands(cmd *latest.SyncExecCommand) (string, []string, string, []string) {
	if cmd.Command != "" {
		return cmd.Command, cmd.Args, cmd.Command, cmd.Args
	}

	var (
		onFileChange = cmd.OnFileChange
		onDirCreate  = cmd.OnDirCreate
	)

	if onFileChange == nil {
		onFileChange = &latest.SyncCommand{}
	}
	if onDirCreate == nil {
		onDirCreate = &latest.SyncCommand{}
	}

	return onFileChange.Command, onFileChange.Args, onDirCreate.Command, onDirCreate.Args
}

func parseExcludeFile(path string) ([]string, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "open exclude file")
	}
	defer reader.Close()

	paths, err := dockerignore.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrap(err, "read exclude file")
	}

	return paths, nil
}

func StartStream(client kubectl.Client, pod *v1.Pod, container string, command []string, reader io.Reader, stdoutWriter io.Writer, buffer bool, log logpkg.Logger) error {
	stderrBuffer := &bytes.Buffer{}
	stderrReader, stderrWriter := io.Pipe()
	defer stderrWriter.Close()

	go func() {
		defer stderrReader.Close()

		scanner := scanner.NewScanner(stderrReader)
		for scanner.Scan() {
			log.Info("Helper - " + scanner.Text())
		}
		if scanner.Err() != nil && scanner.Err() != context.Canceled {
			log.Warnf("Helper - Error streaming logs: %v", scanner.Err())
		}
	}()

	var stdErr io.Writer = stderrWriter
	if buffer {
		stdErr = io.MultiWriter(stderrBuffer, stderrWriter)
	}

	err := client.ExecStream(&kubectl.ExecStreamOptions{
		Pod:       pod,
		Container: container,
		Command:   command,
		Stdin:     reader,
		Stdout:    stdoutWriter,
		Stderr:    stdErr,
	})
	if err != nil {
		return fmt.Errorf("%s %v", stderrBuffer.String(), err)
	}
	return nil
}
