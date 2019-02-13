package versions

import (
	"fmt"

	"github.com/covexo/devspace/pkg/devspace/config/versions/config"
	"github.com/covexo/devspace/pkg/devspace/config/versions/latest"
	"github.com/covexo/devspace/pkg/devspace/config/versions/util"
	"github.com/covexo/devspace/pkg/devspace/config/versions/v1alpha1"
)

var versionLoader = map[string]config.New{
	v1alpha1.Version: v1alpha1.New,
	latest.Version:   latest.New,
}

// Parse parses the data into the latest config
func Parse(data map[interface{}]interface{}) (*latest.Config, error) {
	version, ok := data["version"].(string)
	if ok == false {
		return nil, fmt.Errorf("Error parsing config: version not found")
	}

	versionLoadFunc, ok := versionLoader[version]
	if ok == false {
		return nil, fmt.Errorf("Unrecognized config version %s. Please upgrade devspace with `devspace upgrade`", version)
	}

	// Load config
	latestConfig := versionLoadFunc()
	err := util.Convert(data, latestConfig)
	if err != nil {
		return nil, fmt.Errorf("Error converting config: %v", err)
	}

	// Upgrade config to latest
	for latestConfig.GetVersion() != latest.Version {
		upgradedConfig, err := latestConfig.Upgrade()
		if err != nil {
			return nil, fmt.Errorf("Error upgrading config from version %s: %v", latestConfig.GetVersion(), err)
		}

		latestConfig = upgradedConfig
	}

	// Convert
	latestConfigConverted, ok := latestConfig.(*latest.Config)
	if ok == false {
		return nil, fmt.Errorf("Error converting config, latest config is not the latest version")
	}

	return latestConfigConverted, nil
}
