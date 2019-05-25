package services

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"

	"k8s.io/client-go/kubernetes"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/sync"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// StartSyncFromCmd starts a new sync from command
func StartSyncFromCmd(config *latest.Config, client kubernetes.Interface, cmdParameter targetselector.CmdParameter, containerPath string, exclude []string, log log.Logger) error {
	/*var (
		localPath = "."
	)

	absLocalPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("Unable to resolve localSubPath %s: %v", localPath, err)
	}

	targetSelector, err := targetselector.NewTargetSelector(config, &targetselector.SelectorParameter{
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
	Sync := &sync.Sync{
		DevSpaceConfig: config,
		Kubectl:        client,
		Pod:            pod,
		Container:      container,
		WatchPath:      absLocalPath,
		DestPath:       containerPath,
		ExcludePaths:   exclude,
		CustomLog:      log,
		SyncDone:       syncDone,
		Verbose:        false,
	}

	log.Donef("Sync started on %s <-> %s (Pod: %s/%s)", absLocalPath, containerPath, pod.Namespace, pod.Name)

	err = Sync.Start()
	if err != nil {
		log.Fatalf("Sync error: %s", err.Error())
	}

	// Wait till sync is finished
	<-syncDone

	return nil*/
	return nil
}

// StartSync starts the syncing functionality
func StartSync(config *latest.Config, client kubernetes.Interface, verboseSync bool, log log.Logger) ([]*sync.Sync, error) {
	/*if config.Dev.Sync == nil {
		return []*sync.Sync{}, nil
	}

	Syncs := make([]*sync.Sync, 0, len(*config.Dev.Sync))
	for _, syncPath := range *config.Dev.Sync {
		localPath := "."
		if syncPath.LocalSubPath != nil {
			localPath = *syncPath.LocalSubPath
		}

		absLocalPath, err := filepath.Abs(localPath)
		if err != nil {
			return nil, fmt.Errorf("Unable to resolve localSubPath %s: %v", localPath, err)
		}

		selector, err := targetselector.NewTargetSelector(config, &targetselector.SelectorParameter{
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

		Sync := &sync.Sync{
			DevSpaceConfig:            config,
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
			Sync.ExcludePaths = *syncPath.ExcludePaths
		}

		if syncPath.DownloadExcludePaths != nil {
			Sync.DownloadExcludePaths = *syncPath.DownloadExcludePaths
		}

		if syncPath.UploadExcludePaths != nil {
			Sync.UploadExcludePaths = *syncPath.UploadExcludePaths
		}

		if syncPath.BandwidthLimits != nil {
			if syncPath.BandwidthLimits.Download != nil {
				Sync.DownstreamLimit = *syncPath.BandwidthLimits.Download * 1024
			}

			if syncPath.BandwidthLimits.Upload != nil {
				Sync.UpstreamLimit = *syncPath.BandwidthLimits.Upload * 1024
			}
		}

		err = Sync.Start()
		if err != nil {
			log.Fatalf("Sync error: %s", err.Error())
		}

		log.Donef("Sync started on %s <-> %s (Pod: %s/%s)", absLocalPath, containerPath, pod.Namespace, pod.Name)

		if syncPath.WaitInitialSync != nil && *syncPath.WaitInitialSync == true {
			log.StartWait("Sync: waiting for intial sync to complete")
			<-Sync.UpstreamInitialSyncDone
			<-Sync.DownstreamInitialSyncDone
			log.StopWait()
		}

		Syncs = append(Syncs, Sync)
	}

	return Syncs, nil*/
	return nil, nil
}
