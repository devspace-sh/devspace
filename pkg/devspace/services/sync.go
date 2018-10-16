package services

import (
	"fmt"
	"path/filepath"
	"strings"

	"k8s.io/client-go/kubernetes"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/devspace/sync"
	"github.com/covexo/devspace/pkg/util/log"
)

// StartSync starts the syncing functionality
func StartSync(client *kubernetes.Clientset, verboseSync bool, log log.Logger) ([]*sync.SyncConfig, error) {
	config := configutil.GetConfig()
	syncConfigs := make([]*sync.SyncConfig, 0, len(*config.DevSpace.Sync))

	for _, syncPath := range *config.DevSpace.Sync {
		absLocalPath, err := filepath.Abs(*syncPath.LocalSubPath)
		if err != nil {
			return nil, fmt.Errorf("Unable to resolve localSubPath %s: %v", *syncPath.LocalSubPath, err)
		}

		// Retrieve pod from label selector
		labels := make([]string, 0, len(*syncPath.LabelSelector))
		for key, value := range *syncPath.LabelSelector {
			labels = append(labels, key+"="+*value)
		}

		// Init namespace
		namespace := ""
		if syncPath.Namespace != nil {
			namespace = *syncPath.Namespace
		}

		pod, err := kubectl.GetNewestRunningPod(client, strings.Join(labels, ", "), namespace)
		if err != nil {
			return nil, fmt.Errorf("Unable to list devspace pods: %v", err)
		} else if pod != nil {
			if len(pod.Spec.Containers) == 0 {
				log.Warnf("Cannot start sync on pod, because selected pod %s/%s has no containers", pod.Namespace, pod.Name)
				continue
			}

			container := &pod.Spec.Containers[0]
			if syncPath.ContainerName != nil && *syncPath.ContainerName != "" {
				found := false

				for _, c := range pod.Spec.Containers {
					if c.Name == *syncPath.ContainerName {
						container = &c
						found = true
						break
					}
				}

				if found == false {
					log.Warnf("Couldn't start sync, because container %s wasn't found in pod %s/%s", *syncPath.ContainerName, pod.Namespace, pod.Name)
					continue
				}
			}

			syncConfig := &sync.SyncConfig{
				Kubectl:   client,
				Pod:       pod,
				Container: container,
				WatchPath: absLocalPath,
				DestPath:  *syncPath.ContainerPath,
				Verbose:   verboseSync,
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

			err = syncConfig.Start()
			if err != nil {
				log.Fatalf("Sync error: %s", err.Error())
			}

			log.Donef("Sync started on %s <-> %s (Pod: %s/%s)", absLocalPath, *syncPath.ContainerPath, pod.Namespace, pod.Name)
			syncConfigs = append(syncConfigs, syncConfig)
		}
	}

	return syncConfigs, nil
}
