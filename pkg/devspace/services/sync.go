package services

import (
	"fmt"
	"path/filepath"

	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"

	"k8s.io/client-go/kubernetes"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/sync"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// StartSyncFromCmd starts a new sync from command
func StartSyncFromCmd(client *kubernetes.Clientset, cmdParameter targetselector.CmdParameter, containerPath string, exclude []string, log log.Logger) error {
	var (
		localPath = "."
	)

	absLocalPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("Unable to resolve localSubPath %s: %v", localPath, err)
	}

	targetSelector, err := targetselector.NewTargetSelector(&targetselector.SelectorParameter{
		CmdParameter: cmdParameter,
	}, true)
	if err != nil {
		return err
	}

	pod, container, err := targetSelector.GetContainer(client)
	if err != nil {
		return err
	}

	if containerPath == "" {
		containerPath = "."
	}

	syncDone := make(chan bool)
	syncConfig := &sync.SyncConfig{
		Kubectl:      client,
		Pod:          pod,
		Container:    container,
		WatchPath:    absLocalPath,
		DestPath:     containerPath,
		ExcludePaths: exclude,
		CustomLog:    log,
		SyncDone:     syncDone,
		Verbose:      false,
	}

	log.Donef("Sync started on %s <-> %s (Pod: %s/%s)", absLocalPath, containerPath, pod.Namespace, pod.Name)

	err = syncConfig.Start()
	if err != nil {
		log.Fatalf("Sync error: %s", err.Error())
	}

	// Wait till sync is finished
	<-syncDone

	return nil
}

// StartSync starts the syncing functionality
func StartSync(client *kubernetes.Clientset, verboseSync bool, log log.Logger) ([]*sync.SyncConfig, error) {
	config := configutil.GetConfig()
	if config.Dev.Sync == nil {
		return []*sync.SyncConfig{}, nil
	}

	syncConfigs := make([]*sync.SyncConfig, 0, len(*config.Dev.Sync))
	for _, syncPath := range *config.Dev.Sync {
		localPath := "."
		if syncPath.LocalSubPath != nil {
			localPath = *syncPath.LocalSubPath
		}

		absLocalPath, err := filepath.Abs(localPath)
		if err != nil {
			return nil, fmt.Errorf("Unable to resolve localSubPath %s: %v", localPath, err)
		}

		selector, err := targetselector.NewTargetSelector(&targetselector.SelectorParameter{
			ConfigParameter: targetselector.ConfigParameter{
				Selector:      syncPath.Selector,
				Namespace:     syncPath.Namespace,
				LabelSelector: syncPath.LabelSelector,
				ContainerName: syncPath.ContainerName,
			},
		}, false)
		if err != nil {
			return nil, fmt.Errorf("Error creating target selector: %v", err)
		}

		log.StartWait("Sync: Waiting for pods...")
		pod, container, err := selector.GetContainer(client)
		log.StopWait()
		if err != nil {
			return nil, fmt.Errorf("Unable to start sync, because an error occured during pod selection: %v", err)
		}

		containerPath := "."
		if syncPath.ContainerPath != nil {
			containerPath = *syncPath.ContainerPath
		}

		var upstreamInitialSyncDone chan bool
		var downstreamInitialSyncDone chan bool

		if syncPath.WaitInitialSync != nil && *syncPath.WaitInitialSync == true {
			upstreamInitialSyncDone = make(chan bool)
			downstreamInitialSyncDone = make(chan bool)
		}

		syncConfig := &sync.SyncConfig{
			Kubectl:                   client,
			Pod:                       pod,
			Container:                 container,
			WatchPath:                 absLocalPath,
			DestPath:                  containerPath,
			Verbose:                   verboseSync,
			UpstreamInitialSyncDone:   upstreamInitialSyncDone,
			DownstreamInitialSyncDone: downstreamInitialSyncDone,
		}

		if syncPath.ExcludePaths != nil {
			syncConfig.ExcludePaths = *syncPath.ExcludePaths
		}

		if syncPath.DownloadExcludePaths != nil {
			syncConfig.DownloadExcludePaths = *syncPath.DownloadExcludePaths
		}

		if syncPath.UploadExcludePaths != nil {
			syncConfig.UploadExcludePaths = *syncPath.UploadExcludePaths
		}

		if syncPath.BandwidthLimits != nil {
			if syncPath.BandwidthLimits.Download != nil {
				syncConfig.DownstreamLimit = *syncPath.BandwidthLimits.Download * 1024
			}

			if syncPath.BandwidthLimits.Upload != nil {
				syncConfig.UpstreamLimit = *syncPath.BandwidthLimits.Upload * 1024
			}
		}

		err = syncConfig.Start()
		if err != nil {
			log.Fatalf("Sync error: %s", err.Error())
		}

		log.Donef("Sync started on %s <-> %s (Pod: %s/%s)", absLocalPath, containerPath, pod.Namespace, pod.Name)

		if syncPath.WaitInitialSync != nil && *syncPath.WaitInitialSync == true {
			log.StartWait("Sync: waiting for intial sync to complete")
			<-syncConfig.UpstreamInitialSyncDone
			<-syncConfig.DownstreamInitialSyncDone
			log.StopWait()
		}

		syncConfigs = append(syncConfigs, syncConfig)
	}

	return syncConfigs, nil
}
