package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/devspace/sync"
	"github.com/devspace-cloud/devspace/pkg/devspace/upgrade"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

// SyncHelperBaseURL is the base url where to look for the sync helper
const SyncHelperBaseURL = "https://github.com/devspace-cloud/devspace/releases"

// SyncHelperTempFolder is the local folder where we store the sync helper
const SyncHelperTempFolder = "sync"

// SyncBinaryRegEx is the regexp that finds the correct download link for the sync helper binary
var SyncBinaryRegEx = regexp.MustCompile(`href="(\/devspace-cloud\/devspace\/releases\/download\/[^\/]*\/sync)"`)

// SyncHelperContainerPath is the path of the sync helper in the container
const SyncHelperContainerPath = "/tmp/sync"

// StartSyncFromCmd starts a new sync from command
func (serviceClient *client) StartSyncFromCmd(localPath, containerPath string, exclude []string, verbose, downloadOnInitialSync, waitInitialSync bool) error {
	targetSelector, err := targetselector.NewTargetSelector(serviceClient.config, serviceClient.client, serviceClient.selectorParameter, true, nil)
	if err != nil {
		return err
	}

	pod, container, err := targetSelector.GetContainer(false, serviceClient.log)
	if err != nil {
		return err
	}

	if containerPath == "" {
		containerPath = "."
	}
	if localPath == "" {
		localPath = "."
	}

	syncDone := make(chan bool)
	syncConfig := &latest.SyncConfig{
		LocalSubPath:          localPath,
		ContainerPath:         containerPath,
		DownloadOnInitialSync: &downloadOnInitialSync,
		WaitInitialSync:       &waitInitialSync,
	}
	if len(exclude) > 0 {
		syncConfig.ExcludePaths = exclude
	}

	serviceClient.log.StartWait("Starting sync...")
	syncClient, err := serviceClient.startSync(pod, container.Name, syncConfig, verbose, syncDone, serviceClient.log)
	serviceClient.log.StopWait()
	if err != nil {
		return errors.Wrap(err, "start sync")
	}

	err = syncClient.Start()
	if err != nil {
		return errors.Errorf("Sync error: %v", err)
	}

	serviceClient.log.Donef("Sync started on %s <-> %s (Pod: %s/%s)", syncClient.LocalPath, containerPath, pod.Namespace, pod.Name)

	if waitInitialSync {
		serviceClient.log.StartWait("Sync: waiting for intial sync to complete")
		<-syncClient.Options.UpstreamInitialSyncDone
		<-syncClient.Options.DownstreamInitialSyncDone
		serviceClient.log.StopWait()

		return nil
	}

	// Wait till sync is finished
	<-syncDone

	return nil
}

// StartSync starts the syncing functionality
func (serviceClient *client) StartSync(verboseSync bool) ([]*sync.Sync, error) {
	if serviceClient.config.Dev.Sync == nil {
		return []*sync.Sync{}, nil
	}

	syncClients := make([]*sync.Sync, 0, len(serviceClient.config.Dev.Sync))
	for _, syncConfig := range serviceClient.config.Dev.Sync {
		var imageSelector []string
		if syncConfig.ImageName != "" {
			imageConfigCache := serviceClient.generated.GetActive().GetImageCache(syncConfig.ImageName)
			if imageConfigCache.ImageName != "" {
				imageSelector = []string{imageConfigCache.ImageName + ":" + imageConfigCache.Tag}
			}
		}

		selector, err := targetselector.NewTargetSelector(serviceClient.config, serviceClient.client, &targetselector.SelectorParameter{
			ConfigParameter: targetselector.ConfigParameter{
				Namespace:     syncConfig.Namespace,
				LabelSelector: syncConfig.LabelSelector,
				ContainerName: syncConfig.ContainerName,
			},
		}, false, imageSelector)
		if err != nil {
			return nil, errors.Errorf("Error creating target selector: %v", err)
		}

		serviceClient.log.StartWait("Sync: Waiting for pods...")
		pod, container, err := selector.GetContainer(false, serviceClient.log)
		serviceClient.log.StopWait()
		if err != nil {
			return nil, errors.Errorf("Error selecting pod: %v", err)
		}

		serviceClient.log.StartWait("Starting sync...")
		syncClient, err := serviceClient.startSync(pod, container.Name, syncConfig, verboseSync, nil, nil)
		serviceClient.log.StopWait()
		if err != nil {
			return nil, errors.Wrap(err, "start sync")
		}

		err = syncClient.Start()
		if err != nil {
			return nil, errors.Errorf("Sync error: %v", err)
		}

		containerPath := "."
		if syncConfig.ContainerPath != "" {
			containerPath = syncConfig.ContainerPath
		}

		serviceClient.log.Donef("Sync started on %s <-> %s (Pod: %s/%s)", syncClient.LocalPath, containerPath, pod.Namespace, pod.Name)
		if syncConfig.WaitInitialSync != nil && *syncConfig.WaitInitialSync == true {
			serviceClient.log.StartWait("Sync: waiting for intial sync to complete")
			<-syncClient.Options.UpstreamInitialSyncDone
			<-syncClient.Options.DownstreamInitialSyncDone
			serviceClient.log.StopWait()
		}

		syncClients = append(syncClients, syncClient)
	}

	return syncClients, nil
}

func (serviceClient *client) startSync(pod *v1.Pod, container string, syncConfig *latest.SyncConfig, verbose bool, syncDone chan bool, customLog log.Logger) (*sync.Sync, error) {
	err := serviceClient.injectSync(pod, container)
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

	downloadOnInitialSync := false
	if syncConfig.DownloadOnInitialSync != nil {
		downloadOnInitialSync = *syncConfig.DownloadOnInitialSync
	}

	options := &sync.Options{
		Verbose:               verbose,
		SyncDone:              syncDone,
		DownloadOnInitialSync: downloadOnInitialSync,
		Log:                   customLog,
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

	if syncConfig.WaitInitialSync != nil && *syncConfig.WaitInitialSync == true {
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

	syncClient, err := sync.NewSync(localPath, options)
	if err != nil {
		return nil, errors.Wrap(err, "create sync")
	}

	// Start upstream
	upstreamArgs := []string{SyncHelperContainerPath, "--upstream"}
	for _, exclude := range options.ExcludePaths {
		upstreamArgs = append(upstreamArgs, "--exclude", exclude)
	}
	for _, exclude := range options.DownloadExcludePaths {
		upstreamArgs = append(upstreamArgs, "--exclude", exclude)
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

	go serviceClient.startStream(syncClient, pod, container, upstreamArgs, upStdinReader, upStdoutWriter)

	err = syncClient.InitUpstream(upStdoutReader, upStdinWriter)
	if err != nil {
		return nil, errors.Wrap(err, "init upstream")
	}

	// Start downstream
	downstreamArgs := []string{SyncHelperContainerPath, "--downstream"}
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

	go serviceClient.startStream(syncClient, pod, container, downstreamArgs, downStdinReader, downStdoutWriter)

	err = syncClient.InitDownstream(downStdoutReader, downStdinWriter)
	if err != nil {
		return nil, errors.Wrap(err, "init downstream")
	}

	return syncClient, nil
}

func (serviceClient *client) startStream(syncClient *sync.Sync, pod *v1.Pod, container string, command []string, reader io.Reader, writer io.Writer) {
	stderrBuffer := &bytes.Buffer{}

	err := serviceClient.client.ExecStream(pod, container, command, false, reader, writer, stderrBuffer)
	if err != nil {
		syncClient.Stop(errors.Errorf("Sync - connection lost to pod %s/%s: %s %v", pod.Namespace, pod.Name, stderrBuffer.String(), err))
	}
}

func (serviceClient *client) injectSync(pod *v1.Pod, container string) error {
	// Compare sync versions
	version := upgrade.GetRawVersion()
	if version == "" {
		version = "latest"
	}

	// Check if sync is already in pod
	stdout, _, err := serviceClient.client.ExecBuffered(pod, container, []string{"/tmp/sync", "--version"}, nil)
	if err != nil || version != string(stdout) {
		homedir, err := homedir.Dir()
		if err != nil {
			return err
		}

		syncBinaryFolder := filepath.Join(homedir, constants.DefaultHomeDevSpaceFolder, SyncHelperTempFolder, version)
		filepath := filepath.Join(syncBinaryFolder, "sync")

		// Download sync helper if necessary
		err = serviceClient.downloadSyncHelper(filepath, syncBinaryFolder, version)
		if err != nil {
			return errors.Wrap(err, "download sync helper")
		}

		// Inject sync helper
		err = serviceClient.injectSyncHelper(pod, container, filepath)
		if err != nil {
			return errors.Wrap(err, "inject sync helper")
		}
	}

	return nil
}

func (serviceClient *client) downloadSyncHelper(filepath, syncBinaryFolder, version string) error {
	// Check if file exists
	_, err := os.Stat(filepath)
	if err == nil {
		return nil
	}

	// Make sync binary
	err = os.MkdirAll(syncBinaryFolder, 0755)
	if err != nil {
		return errors.Wrap(err, "mkdir sync binary folder")
	}

	return serviceClient.downloadFile(version, filepath)
}

func (serviceClient *client) downloadFile(version string, filepath string) error {
	// Create download url
	url := ""
	if version == "latest" {
		url = fmt.Sprintf("%s/%s", SyncHelperBaseURL, version)
	} else {
		url = fmt.Sprintf("%s/tag/%s", SyncHelperBaseURL, version)
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

	matches := SyncBinaryRegEx.FindStringSubmatch(string(body))
	if len(matches) != 2 {
		return errors.Errorf("Couldn't find sync helper in github release %s at url %s", version, url)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return errors.Wrap(err, "create filepath")
	}
	defer out.Close()

	resp, err = http.Get("https://github.com" + matches[1])
	if err != nil {
		return errors.Wrap(err, "download sync helper")
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return errors.Wrap(err, "download sync helper to file")
	}

	return nil
}

func (serviceClient *client) injectSyncHelper(pod *v1.Pod, container string, filepath string) error {
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
		errChan <- serviceClient.client.CopyFromReader(pod, container, "/tmp", reader)
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

	hdr.Name = "sync"

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
