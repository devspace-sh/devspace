package configure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
)

// AddSyncPath adds a new sync path to the config
func AddSyncPath(localPath, containerPath, namespace, labelSelector, excludedPathsString string) error {
	config := configutil.GetBaseConfig(context.Background())

	if config.Dev == nil {
		config.Dev = &latest.DevConfig{}
	}
	if config.Dev.Sync == nil {
		config.Dev.Sync = []*latest.SyncConfig{}
	}

	var labelSelectorMap map[string]string
	var err error

	if labelSelector == "" {
		labelSelector = "app.kubernetes.io/component=" + GetNameOfFirstDeployment(config)
	}

	if labelSelectorMap == nil {
		labelSelectorMap, err = parseSelectors(labelSelector)
		if err != nil {
			return fmt.Errorf("Error parsing selectors: %s", err.Error())
		}
	}

	excludedPaths := make([]string, 0, 0)
	if excludedPathsString != "" {
		excludedPathStrings := strings.Split(excludedPathsString, ",")

		for _, v := range excludedPathStrings {
			excludedPath := strings.TrimSpace(v)
			excludedPaths = append(excludedPaths, excludedPath)
		}
	}

	workdir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Unable to determine current workdir: %s", err.Error())
	}

	localPath = strings.TrimPrefix(localPath, workdir)

	if containerPath[0] != '/' {
		return errors.New("ContainerPath (--container) must start with '/'. Info: There is an issue with MINGW based terminals like git bash")
	}

	Sync := append(config.Dev.Sync, &latest.SyncConfig{
		LabelSelector: labelSelectorMap,
		ContainerPath: containerPath,
		LocalSubPath:  localPath,
		ExcludePaths:  excludedPaths,
		Namespace:     namespace,
	})

	config.Dev.Sync = Sync

	err = configutil.SaveLoadedConfig()
	if err != nil {
		return fmt.Errorf("Couldn't save config file: %s", err.Error())
	}

	return nil
}

// RemoveSyncPath removes a sync path from the config
func RemoveSyncPath(removeAll bool, localPath, containerPath, labelSelector string) error {
	config := configutil.GetBaseConfig(context.Background())
	labelSelectorMap, err := parseSelectors(labelSelector)

	if err != nil {
		return fmt.Errorf("Error parsing selectors: %v", err)
	}

	if len(labelSelectorMap) == 0 && removeAll == false && localPath == "" && containerPath == "" {
		return fmt.Errorf("You have to specify at least one of the supported flags")
	}

	if config.Dev.Sync != nil && len(config.Dev.Sync) > 0 {
		newSyncPaths := make([]*latest.SyncConfig, 0, len(config.Dev.Sync)-1)

		for _, v := range config.Dev.Sync {
			if removeAll ||
				localPath == v.LocalSubPath ||
				containerPath == v.ContainerPath ||
				areLabelMapsEqual(labelSelectorMap, v.LabelSelector) {
				continue
			}

			newSyncPaths = append(newSyncPaths, v)
		}

		config.Dev.Sync = newSyncPaths

		err = configutil.SaveLoadedConfig()
		if err != nil {
			return fmt.Errorf("Couldn't save config file: %v", err)
		}
	}

	return nil
}

func parseSelectors(selectorString string) (map[string]string, error) {
	selectorMap := make(map[string]string)

	if selectorString == "" {
		return selectorMap, nil
	}

	selectors := strings.Split(selectorString, ",")

	for _, v := range selectors {
		keyValue := strings.Split(v, "=")

		if len(keyValue) != 2 {
			return nil, fmt.Errorf("Wrong selector format: %s", selectorString)
		}
		labelSelector := strings.TrimSpace(keyValue[1])
		selectorMap[strings.TrimSpace(keyValue[0])] = labelSelector
	}

	return selectorMap, nil
}

func areLabelMapsEqual(map1 map[string]string, map2 map[string]string) bool {
	if len(map1) != len(map2) {
		return false
	}

	for map1Index, map1Value := range map1 {
		if map2Value, map2Contains := map2[map1Index]; !map2Contains || map2Value != map1Value {
			return false
		}
	}

	return true
}
