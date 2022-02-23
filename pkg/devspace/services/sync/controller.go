package sync

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/inject"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/devspace/sync"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/scanner"
	"github.com/moby/buildkit/frontend/dockerfile/dockerignore"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

type Controller interface {
	Start(ctx *devspacecontext.Context, options *Options) error
}

func NewController() Controller {
	return &controller{}
}

type controller struct{}

type Options struct {
	Name       string
	SyncConfig *latest.SyncConfig
	Arch       string
	Selector   targetselector.TargetSelector

	Done chan struct{}

	RestartOnError bool
	SyncLog        logpkg.Logger

	Verbose bool
}

func (c *controller) Start(ctx *devspacecontext.Context, options *Options) error {
	pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
		"sync_config": options.SyncConfig,
	}, hook.EventsForSingle("start:sync", options.Name).With("sync.start")...)
	if pluginErr != nil {
		return pluginErr
	}

	err := c.startWithWait(ctx, options)
	if err != nil {
		pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
			"sync_config": options.SyncConfig,
			"ERROR":       err,
		}, hook.EventsForSingle("error:sync", options.Name).With("sync.error")...)
		if pluginErr != nil {
			return pluginErr
		}

		return err
	}

	return nil
}

func (c *controller) startWithWait(ctx *devspacecontext.Context, options *Options) error {
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
		pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
			"sync_config": options.SyncConfig,
		}, hook.EventsForSingle("before:initialSync", options.Name).With("sync.beforeInitialSync")...)
		if pluginErr != nil {
			return pluginErr
		}
	}

	// start the sync
	client, pod, err := c.startSync(ctx, options, onInitUploadDone, onInitDownloadDone, onDone, onError)
	if err != nil {
		pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
			"sync_config": options.SyncConfig,
			"ERROR":       err,
		}, hook.EventsForSingle("error:initialSync", options.Name).With("sync.errorInitialSync")...)
		if pluginErr != nil {
			return pluginErr
		}

		return err
	}

	// should wait for initial sync?
	if options.SyncConfig.WaitInitialSync == nil || *options.SyncConfig.WaitInitialSync {
		ctx.Log.Info("Waiting for initial sync to complete")
		var (
			uploadDone   = false
			downloadDone = false
		)
		started := time.Now()
		for {
			select {
			case err := <-onError:
				pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
					"sync_config": options.SyncConfig,
					"ERROR":       err,
				}, hook.EventsForSingle("error:initialSync", options.Name).With("sync.errorInitialSync")...)
				if pluginErr != nil {
					return pluginErr
				}
				return errors.Wrap(err, "initial sync")
			case <-onInitUploadDone:
				uploadDone = true
			case <-onInitDownloadDone:
				downloadDone = true
			case <-ctx.Context.Done():
				client.Stop(nil)
				pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
					"sync_config": options.SyncConfig,
				}, hook.EventsForSingle("stop:sync", options.Name).With("sync.stop")...)
				if pluginErr != nil {
					return pluginErr
				}
				return nil
			case <-onDone:
				if options.Done != nil {
					close(options.Done)
				}
				pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
					"sync_config": options.SyncConfig,
				}, hook.EventsForSingle("stop:sync", options.Name).With("sync.stop")...)
				if pluginErr != nil {
					return pluginErr
				}
				return nil
			}
			if uploadDone && downloadDone {
				ctx.Log.Debugf("Initial sync took: %s", time.Since(started))
				break
			}
		}
		pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
			"sync_config": options.SyncConfig,
		}, hook.EventsForSingle("after:initialSync", options.Name).With("sync.afterInitialSync")...)
		if pluginErr != nil {
			return pluginErr
		}
	}

	// should we restart the client on error?
	if options.RestartOnError {
		go func(syncClient *sync.Sync, options *Options) {
			select {
			case err = <-onError:
				hook.LogExecuteHooks(ctx.WithLogger(options.SyncLog), map[string]interface{}{
					"sync_config": options.SyncConfig,
					"ERROR":       err,
				}, hook.EventsForSingle("restart:sync", options.Name).With("sync.restart")...)

				options.SyncLog.Info("Restarting sync...")
				PrintPodError(ctx.Context, ctx.KubeClient, pod.Pod, options.SyncLog)
				for {
					err := c.startWithWait(ctx.WithLogger(options.SyncLog), options)
					if err != nil {
						hook.LogExecuteHooks(ctx.WithLogger(options.SyncLog), map[string]interface{}{
							"sync_config": options.SyncConfig,
							"ERROR":       err,
						}, hook.EventsForSingle("restart:sync", options.Name).With("sync.restart")...)
						options.SyncLog.Errorf("Error restarting sync: %v", err)
						options.SyncLog.Errorf("Will try again in 15 seconds")
						time.Sleep(time.Second * 15)
						continue
					}

					break
				}
			case <-ctx.Context.Done():
				syncClient.Stop(nil)
				if options.Done != nil {
					close(options.Done)
				}
				hook.LogExecuteHooks(ctx.WithLogger(options.SyncLog), map[string]interface{}{
					"sync_config": options.SyncConfig,
				}, hook.EventsForSingle("stop:sync", options.Name).With("sync.stop")...)
			case <-onDone:
				if options.Done != nil {
					close(options.Done)
				}
				hook.LogExecuteHooks(ctx.WithLogger(options.SyncLog), map[string]interface{}{
					"sync_config": options.SyncConfig,
				}, hook.EventsForSingle("stop:sync", options.Name).With("sync.stop")...)
			}
		}(client, options)
	}

	return nil
}

func PrintPodError(ctx context.Context, kubeClient kubectl.Client, pod *v1.Pod, log logpkg.Logger) {
	// check if pod still exists
	newPod, err := kubeClient.KubeClient().CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			log.Errorf("Restarted because old pod %s/%s seems to be erased", pod.Namespace, pod.Name)
			return
		}

		return
	}

	podStatus := kubectl.GetPodStatus(newPod)
	if podStatus != "Running" {
		log.Errorf("Restarted because old pod %s/%s has status %s", pod.Namespace, pod.Name, podStatus)
	}
}

func (c *controller) startSync(ctx *devspacecontext.Context, options *Options, onInitUploadDone chan struct{}, onInitDownloadDone chan struct{}, onDone chan struct{}, onError chan error) (*sync.Sync, *selector.SelectedPodContainer, error) {
	var (
		syncConfig = options.SyncConfig
	)

	container, err := options.Selector.SelectSingleContainer(ctx.Context, ctx.KubeClient, ctx.Log)
	if err != nil {
		return nil, nil, errors.Errorf("Error selecting pod: %v", err)
	}

	ctx.Log.Info("Starting sync...")
	syncClient, err := c.initClient(ctx, container.Pod, options.Arch, container.Container.Name, syncConfig, options.Verbose, options.SyncLog)
	if err != nil {
		return nil, nil, errors.Wrap(err, "start sync")
	}

	err = syncClient.Start(onInitUploadDone, onInitDownloadDone, onDone, onError)
	if err != nil {
		return nil, nil, errors.Errorf("Sync error: %v", err)
	}

	containerPath := "."
	if syncConfig.ContainerPath != "" {
		containerPath = syncConfig.ContainerPath
	}

	ctx.Log.Donef("Sync started on %s:%s", syncClient.LocalPath, containerPath)
	return syncClient, container, nil
}

func (c *controller) initClient(ctx *devspacecontext.Context, pod *v1.Pod, arch, container string, syncConfig *latest.SyncConfig, verbose bool, customLog logpkg.Logger) (*sync.Sync, error) {
	localPath := "."
	if syncConfig.LocalSubPath != "" {
		localPath = syncConfig.LocalSubPath
	}

	// make sure we resolve it correctly
	localPath = ctx.ResolvePath(localPath)

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

	err = inject.InjectDevSpaceHelper(ctx.KubeClient, pod, container, string(arch), customLog)
	if err != nil {
		return nil, err
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
		ResolveCommand: func(command string, args []string) (string, []string, error) {
			return hook.ResolveCommand(command, args, ctx.Config, ctx.Dependencies)
		},
	}

	// Initialize log
	if options.Log == nil {
		options.Log = logpkg.GetFileLogger("sync")
	}

	// add exec hooks
	if syncConfig.OnUpload != nil {
		options.Exec = syncConfig.OnUpload.Exec
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
		err := StartStream(ctx.Context, ctx.KubeClient, pod, container, upstreamArgs, upStdinReader, upStdoutWriter, true, options.Log)
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
		err := StartStream(ctx.Context, ctx.KubeClient, pod, container, downstreamArgs, downStdinReader, downStdoutWriter, true, options.Log)
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

func StartStream(ctx context.Context, client kubectl.Client, pod *v1.Pod, container string, command []string, reader io.Reader, stdoutWriter io.Writer, buffer bool, log logpkg.Logger) error {
	stderrBuffer := &bytes.Buffer{}
	stderrReader, stderrWriter := io.Pipe()
	defer stderrWriter.Close()

	go func() {
		defer stderrReader.Close()
		s := scanner.NewScanner(stderrReader)
		for s.Scan() {
			log.Info("Helper - " + s.Text())
		}
		if s.Err() != nil && s.Err() != context.Canceled {
			log.Warnf("Helper - Error streaming logs: %v", s.Err())
		}
	}()

	var stdErr io.Writer = stderrWriter
	if buffer {
		stdErr = io.MultiWriter(stderrBuffer, stderrWriter)
	}

	err := client.ExecStream(ctx, &kubectl.ExecStreamOptions{
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
