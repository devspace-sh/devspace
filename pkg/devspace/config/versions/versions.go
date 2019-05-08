package versions

import (
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1alpha1"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1alpha2"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1alpha3"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1alpha4"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1beta1"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	yaml "gopkg.in/yaml.v2"
)

var versionLoader = map[string]config.New{
	v1alpha1.Version: v1alpha1.New,
	v1alpha2.Version: v1alpha2.New,
	v1alpha3.Version: v1alpha3.New,
	v1alpha4.Version: v1alpha4.New,
	v1beta1.Version:  v1beta1.New,
	latest.Version:   latest.New,
}

// Parse parses the data into the latest config
func Parse(data map[interface{}]interface{}) (*latest.Config, error) {
	version, ok := data["version"].(string)
	if ok == false {
		// This is needed because overrides usually don't have versions
		data["version"] = latest.Version
		version = latest.Version
	}

	versionLoadFunc, ok := versionLoader[version]
	if ok == false {
		return nil, fmt.Errorf("Unrecognized config version %s. Please upgrade devspace with `devspace upgrade`", version)
	}

	// Load config strict
	latestConfig := versionLoadFunc()
	out, err := yaml.Marshal(data)
	if err != nil {
		return nil, err
	}
	err = yaml.UnmarshalStrict(out, latestConfig)
	if err != nil {
		return nil, fmt.Errorf("Error loading config: %v", err)
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

	// Update version to latest
	latestConfigConverted.Version = ptr.String(latest.Version)

	return latestConfigConverted, nil
}
