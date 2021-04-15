package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/loft-sh/devspace/assets"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/loft-sh/devspace/pkg/util/imageselector"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/devspace/sync"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

// DevSpaceHelperBaseURL is the base url where to look for the sync helper
const DevSpaceHelperBaseURL = "https://github.com/loft-sh/devspace/releases"

// DevSpaceHelperTempFolder is the local folder where we store the sync helper
const DevSpaceHelperTempFolder = "devspacehelper"

// helperBinaryRegEx is the regexp that finds the correct download link for the sync helper binary
var helperBinaryRegEx = `href="(\/loft-sh\/devspace\/releases\/download\/[^\/]*\/%s)"`

// DevSpaceHelperContainerPath is the path of the devspace helper in the container
const DevSpaceHelperContainerPath = "/tmp/devspacehelper"

// StartSyncFromCmd starts a new sync from command
func (serviceClient *client) StartSyncFromCmd(targetOptions targetselector.Options, syncConfig *latest.SyncConfig, interrupt chan error, verbose bool) error {
	syncDone := make(chan bool)
	options := &startClientOptions{
		Interrupt: interrupt,

		SyncConfig:    syncConfig,
		TargetOptions: targetOptions,

		RestartOnError: true,
		RestartLog:     serviceClient.log,

		SyncDone: syncDone,
		SyncLog:  serviceClient.log,

		Verbose: verbose,
	}

	err := serviceClient.startSyncClient(options, serviceClient.log)
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
	for idx, syncConfig := range serviceClient.config.Config().Dev.Sync {
		targetOptions := targetselector.NewEmptyOptions().ApplyConfigParameter(syncConfig.LabelSelector, syncConfig.Namespace, syncConfig.ContainerName, "")
		targetOptions.AllowPick = false
		targetOptions.WaitingStrategy = targetselector.NewUntilNewestRunningWaitingStrategy(time.Second * 2)

		// set options
		options := &startClientOptions{
			Interrupt: interrupt,

			SyncConfig:    syncConfig,
			TargetOptions: targetOptions,

			RestartOnError: true,
			RestartLog:     logpkg.Discard,
			Verbose:        verboseSync,
		}

		// should we print the logs?
		if printSyncLog {
			prefix := fmt.Sprintf("[%d:sync] ", idx)
			if syncConfig.ImageName != "" {
				prefix = fmt.Sprintf("[%d:sync:%s] ", idx, syncConfig.ImageName)
			}

			logger := logpkg.NewPrefixLogger(prefix, logpkg.Colors[idx%len(logpkg.Colors)], serviceClient.log)
			options.SyncLog = logger
			options.RestartLog = logger
		}

		err := serviceClient.startSyncClient(options, serviceClient.log)
		if err != nil {
			return errors.Errorf("Unable to start sync: %v", err)
		}
	}

	return nil
}

type startClientOptions struct {
	SyncConfig    *latest.SyncConfig
	TargetOptions targetselector.Options

	Interrupt chan error

	RestartOnError bool
	RestartLog     logpkg.Logger

	SyncDone chan bool
	SyncLog  logpkg.Logger

	Verbose bool
}

func (serviceClient *client) startSyncClient(options *startClientOptions, log logpkg.Logger) error {
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
			return err
		}

		err = os.MkdirAll(localPath, os.ModePerm)
		if err != nil {
			return err
		}
	}

	options.TargetOptions.ImageSelector, err = imageselector.Resolve(syncConfig.ImageName, serviceClient.config, serviceClient.dependencies)
	if err != nil {
		return err
	}

	log.StartWait("Sync: Waiting for pods...")
	container, err := targetselector.NewTargetSelector(serviceClient.client).SelectSingleContainer(context.TODO(), options.TargetOptions, serviceClient.log)
	log.StopWait()
	if err != nil {
		return errors.Errorf("Error selecting pod: %v", err)
	}

	syncDone := make(chan bool)

	log.StartWait("Starting sync...")
	syncClient, err := serviceClient.startSync(container.Pod, container.Container.Name, syncConfig, options.Verbose, syncDone, options.SyncLog)
	log.StopWait()
	if err != nil {
		return errors.Wrap(err, "start sync")
	}

	err = syncClient.Start()
	if err != nil {
		return errors.Errorf("Sync error: %v", err)
	}

	containerPath := "."
	if syncConfig.ContainerPath != "" {
		containerPath = syncConfig.ContainerPath
	}

	log.Donef("Sync started on %s <-> %s (Pod: %s/%s)", syncClient.LocalPath, containerPath, container.Pod.Namespace, container.Pod.Name)

	if syncConfig.WaitInitialSync == nil || *syncConfig.WaitInitialSync == true {
		log.StartWait("Sync: waiting for initial sync to complete")
		<-syncClient.Options.UpstreamInitialSyncDone
		<-syncClient.Options.DownstreamInitialSyncDone
		log.StopWait()
	}

	// Should we restart the client on error?
	if options.RestartOnError {
		go func(syncClient *sync.Sync, options *startClientOptions) {
			select {
			case err = <-syncClient.Options.SyncError:
				if serviceClient.isFatalSyncError(err) {
					serviceClient.log.Fatalf("Fatal error in sync: %v", err)
				}

				options.RestartLog.Info("Restarting sync...")
				for {
					time.Sleep(time.Second * 5)
					err := serviceClient.startSyncClient(options, options.RestartLog)
					if err != nil {
						serviceClient.log.Errorf("Error restarting sync: %v", err)
						serviceClient.log.Errorf("Will try again in 5 seconds")
						continue
					}

					break
				}
			case <-options.Interrupt:
				syncClient.Stop(nil)
			case <-syncDone:
				if options.SyncDone != nil {
					close(options.SyncDone)
				}
			}
		}(syncClient, options)
	}

	return nil
}

func (serviceClient *client) isFatalSyncError(err error) bool {
	if strings.Index(err.Error(), "no such file or directory") != -1 {
		return true
	}
	if strings.Index(err.Error(), "You are trying to sync the complete container root") != -1 {
		return true
	}

	return false
}

func (serviceClient *client) startSync(pod *v1.Pod, container string, syncConfig *latest.SyncConfig, verbose bool, syncDone chan bool, customLog logpkg.Logger) (*sync.Sync, error) {
	err := InjectDevSpaceHelper(serviceClient.client, pod, container, string(syncConfig.Arch), serviceClient.log)
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
		SyncError:            make(chan error),
		SyncDone:             syncDone,
		InitialSyncCompareBy: compareBy,
		InitialSync:          syncConfig.InitialSync,
		UpstreamDisabled:     upstreamDisabled,
		DownstreamDisabled:   downstreamDisabled,
		Log:                  customLog,
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

	if len(syncConfig.DownloadExcludePaths) > 0 {
		options.DownloadExcludePaths = syncConfig.DownloadExcludePaths
	}

	if len(syncConfig.UploadExcludePaths) > 0 {
		options.UploadExcludePaths = syncConfig.UploadExcludePaths
	}

	if syncConfig.WaitInitialSync == nil || *syncConfig.WaitInitialSync == true {
		options.UpstreamInitialSyncDone = make(chan bool)
		options.DownstreamInitialSyncDone = make(chan bool)
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

	syncClient, err := sync.NewSync(localPath, options)
	if err != nil {
		return nil, errors.Wrap(err, "create sync")
	}

	// Start upstream
	upstreamArgs := []string{DevSpaceHelperContainerPath, "sync", "upstream"}
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
		if onUpload.OnBatch != nil && onUpload.OnBatch.Command != "" {
			upstreamArgs = append(upstreamArgs, "--batchcmd", onUpload.OnBatch.Command)
			for _, arg := range onUpload.OnBatch.Args {
				upstreamArgs = append(upstreamArgs, "--batchargs", arg)
			}
		}
	}

	upstreamArgs = append(upstreamArgs, containerPath)

	upStdinReader, upStdinWriter, err := os.Pipe()
	if err != nil {
		return nil, errors.Wrap(err, "create pipe")
	}
	upStdoutReader, upStdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, errors.Wrap(err, "create pipe")
	}

	go func() {
		err := serviceClient.startStream(pod, container, upstreamArgs, upStdinReader, upStdoutWriter)
		if err != nil {
			syncClient.Stop(errors.Errorf("Sync - connection lost to pod %s/%s: %v", pod.Namespace, pod.Name, err))
		}
	}()

	err = syncClient.InitUpstream(upStdoutReader, upStdinWriter)
	if err != nil {
		return nil, errors.Wrap(err, "init upstream")
	}

	// Start downstream
	downstreamArgs := []string{DevSpaceHelperContainerPath, "sync", "downstream"}
	if syncConfig.ThrottleChangeDetection != nil {
		downstreamArgs = append(downstreamArgs, "--throttle", strconv.FormatInt(*syncConfig.ThrottleChangeDetection, 10))
	}
	for _, exclude := range options.ExcludePaths {
		downstreamArgs = append(downstreamArgs, "--exclude", exclude)
	}
	for _, exclude := range options.DownloadExcludePaths {
		downstreamArgs = append(downstreamArgs, "--exclude", exclude)
	}
	downstreamArgs = append(downstreamArgs, containerPath)

	downStdinReader, downStdinWriter, err := os.Pipe()
	if err != nil {
		return nil, errors.Wrap(err, "create pipe")
	}
	downStdoutReader, downStdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, errors.Wrap(err, "create pipe")
	}

	go func() {
		err := serviceClient.startStream(pod, container, downstreamArgs, downStdinReader, downStdoutWriter)
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

func (serviceClient *client) startStream(pod *v1.Pod, container string, command []string, reader io.Reader, writer io.Writer) error {
	stderrBuffer := &bytes.Buffer{}
	err := serviceClient.client.ExecStream(&kubectl.ExecStreamOptions{
		Pod:       pod,
		Container: container,
		Command:   command,
		Stdin:     reader,
		Stdout:    writer,
		Stderr:    stderrBuffer,
	})
	if err != nil {
		return fmt.Errorf("%s %v", stderrBuffer.String(), err)
	}
	return nil
}

func InjectDevSpaceHelper(client kubectl.Client, pod *v1.Pod, container string, arch string, log logpkg.Logger) error {
	// Compare sync versions
	version := upgrade.GetRawVersion()
	if version == "" {
		version = "latest"
	}
	if arch != "" {
		if latest.ContainerArchitecture(arch) == latest.ContainerArchitectureAmd64 {
			arch = ""
		} else {
			arch = "-" + arch
		}
	}

	// Check if sync is already in pod
	localHelperName := "devspacehelper" + arch
	stdout, _, err := client.ExecBuffered(pod, container, []string{DevSpaceHelperContainerPath, "version"}, nil)
	if err != nil || version != string(stdout) {
		// check if we can find it in the assets
		helperBytes, err := assets.Asset("release/" + localHelperName)
		if err == nil {
			return injectSyncHelperFromBytes(client, pod, container, helperBytes)
		}

		homedir, err := homedir.Dir()
		if err != nil {
			return err
		}

		syncBinaryFolder := filepath.Join(homedir, constants.DefaultHomeDevSpaceFolder, DevSpaceHelperTempFolder, version)

		// Download sync helper if necessary
		err = downloadSyncHelper(localHelperName, syncBinaryFolder, version, log)
		if err != nil {
			return errors.Wrap(err, "download devspace helper")
		}

		// Inject sync helper
		filepath := filepath.Join(syncBinaryFolder, localHelperName)
		err = injectSyncHelper(client, pod, container, filepath)
		if err != nil {
			return errors.Wrap(err, "inject devspace helper")
		}
	}

	return nil
}

func downloadSyncHelper(helperName, syncBinaryFolder, version string, log logpkg.Logger) error {
	filepath := filepath.Join(syncBinaryFolder, helperName)

	// Check if file exists
	_, err := os.Stat(filepath)
	if err == nil {
		// make sure the sha is correct, but skip for latest because that is development
		if version == "latest" {
			return nil
		}

		// download sha256 html
		url := fmt.Sprintf("https://github.com/loft-sh/devspace/releases/download/%s/%s.sha256", version, helperName)
		resp, err := http.Get(url)
		if err != nil {
			log.Warnf("Couldn't retrieve helper sha256: %v", err)
			return nil
		}

		shaHash, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Warnf("Couldn't read helper sha256 request: %v", err)
			return nil
		}

		// hash the local binary
		fileHash, err := hash.File(filepath)
		if err != nil {
			log.Warnf("Couldn't hash local helper binary: %v", err)
			return nil
		}

		// the file is correct we skip downloading
		if fileHash == strings.Split(string(shaHash), " ")[0] {
			return nil
		}

		// remove the old binary
		err = os.Remove(filepath)
		if err != nil {
			return errors.Wrap(err, "remove corrupt helper binary")
		}
	}

	// Make sync binary
	log.Infof("Couldn't find %s, will try to download it now", helperName)
	err = os.MkdirAll(syncBinaryFolder, 0755)
	if err != nil {
		return errors.Wrap(err, "mkdir helper binary folder")
	}

	return downloadFile(version, filepath, helperName)
}

func downloadFile(version string, filepath string, filename string) error {
	// Create download url
	url := ""
	if version == "latest" {
		url = fmt.Sprintf("%s/%s", DevSpaceHelperBaseURL, version)
	} else {
		url = fmt.Sprintf("%s/tag/%s", DevSpaceHelperBaseURL, version)
	}

	// Download html
	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "get url")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "read body")
	}

	regEx, err := regexp.Compile(fmt.Sprintf(helperBinaryRegEx, filename))
	if err != nil {
		return err
	}

	matches := regEx.FindStringSubmatch(string(body))
	if len(matches) != 2 {
		return errors.Errorf("couldn't find %s in github release %s at url %s", filename, version, url)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return errors.Wrap(err, "create filepath")
	}
	defer out.Close()

	resp, err = http.Get("https://github.com" + matches[1])
	if err != nil {
		return errors.Wrap(err, "download devspace helper")
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return errors.Wrap(err, "download devspace helper to file")
	}

	return nil
}

type helperFileInfo []byte

func (h helperFileInfo) Name() string {
	return DevSpaceHelperTempFolder
}
func (h helperFileInfo) Size() int64 {
	return int64(len([]byte(h)))
}
func (h helperFileInfo) Mode() os.FileMode {
	return 0777
}
func (h helperFileInfo) ModTime() time.Time {
	return time.Now()
}
func (h helperFileInfo) IsDir() bool {
	return false
}
func (h helperFileInfo) Sys() interface{} {
	return nil
}

func injectSyncHelperFromBytes(client kubectl.Client, pod *v1.Pod, container string, b []byte) error {
	// Compress the sync helper and then copy it to the container
	reader, writer, err := os.Pipe()
	if err != nil {
		return errors.Wrap(err, "create pipe")
	}

	defer reader.Close()
	defer writer.Close()

	// Start reading on the other end
	errChan := make(chan error)
	go func() {
		errChan <- client.CopyFromReader(pod, container, "/tmp", reader)
	}()

	// Use compression
	gw := gzip.NewWriter(writer)
	defer gw.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	hdr, err := tar.FileInfoHeader(helperFileInfo(b), DevSpaceHelperTempFolder)
	if err != nil {
		return errors.Wrap(err, "create tar file info header")
	}

	hdr.Name = "devspacehelper"

	// Set permissions correctly
	hdr.Mode = 0777
	hdr.Uid = 0
	hdr.Uname = "root"
	hdr.Gid = 0
	hdr.Gname = "root"

	if err := tarWriter.WriteHeader(hdr); err != nil {
		return errors.Wrap(err, "tar write header")
	}

	if _, err := io.Copy(tarWriter, bytes.NewReader(b)); err != nil {
		return errors.Wrap(err, "tar copy file")
	}

	// Close all writers and file
	tarWriter.Close()
	gw.Close()
	writer.Close()

	return <-errChan
}

func injectSyncHelper(client kubectl.Client, pod *v1.Pod, container string, filepath string) error {
	// Compress the sync helper and then copy it to the container
	reader, writer, err := os.Pipe()
	if err != nil {
		return errors.Wrap(err, "create pipe")
	}

	defer reader.Close()
	defer writer.Close()

	// Start reading on the other end
	errChan := make(chan error)
	go func() {
		errChan <- client.CopyFromReader(pod, container, "/tmp", reader)
	}()

	// Use compression
	gw := gzip.NewWriter(writer)
	defer gw.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	// Stat sync helper
	stat, err := os.Stat(filepath)
	if err != nil {
		return errors.Wrap(err, "stat sync helper")
	}

	// Open file
	f, err := os.Open(filepath)
	if err != nil {
		return errors.Wrap(err, "open file")
	}

	defer f.Close()

	hdr, err := tar.FileInfoHeader(stat, filepath)
	if err != nil {
		return errors.Wrap(err, "create tar file info header")
	}

	hdr.Name = "devspacehelper"

	// Set permissions correctly
	hdr.Mode = 0777
	hdr.Uid = 0
	hdr.Uname = "root"
	hdr.Gid = 0
	hdr.Gname = "root"

	if err := tarWriter.WriteHeader(hdr); err != nil {
		return errors.Wrap(err, "tar write header")
	}

	if _, err := io.Copy(tarWriter, f); err != nil {
		return errors.Wrap(err, "tar copy file")
	}

	// Close all writers and file
	f.Close()
	tarWriter.Close()
	gw.Close()
	writer.Close()

	return <-errChan
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
