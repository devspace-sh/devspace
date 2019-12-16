package versions

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1alpha1"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1alpha2"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1alpha3"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1alpha4"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1beta1"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1beta2"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1beta3"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1beta4"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1beta5"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type loader struct {
	New config.New
}

var versionLoader = map[string]*loader{
	v1alpha1.Version: &loader{New: v1alpha1.New},
	v1alpha2.Version: &loader{New: v1alpha2.New},
	v1alpha3.Version: &loader{New: v1alpha3.New},
	v1alpha4.Version: &loader{New: v1alpha4.New},
	v1beta1.Version:  &loader{New: v1beta1.New},
	v1beta2.Version:  &loader{New: v1beta2.New},
	v1beta3.Version:  &loader{New: v1beta3.New},
	v1beta4.Version:  &loader{New: v1beta4.New},
	v1beta5.Version:  &loader{New: v1beta5.New},
	latest.Version:   &loader{New: latest.New},
}

// ParseProfile loads the base config & a certain profile
func ParseProfile(data map[interface{}]interface{}, profile string) (map[interface{}]interface{}, error) {
	return getProfile(data, profile)
}

// ParseCommands parses only the commands from the config
func ParseCommands(data map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	return getCommands(data)
}

// ParseVariables parses only the variables from the config
func ParseVariables(data map[interface{}]interface{}, log log.Logger) ([]*latest.Variable, error) {
	strippedData, err := getVariables(data)
	if err != nil {
		return nil, errors.Wrap(err, "loading variables")
	}

	config, err := Parse(strippedData, nil, log)
	if err != nil {
		return nil, errors.Wrap(err, "parse variables")
	}

	return config.Vars, nil
}

// Parse parses the data into the latest config
func Parse(data map[interface{}]interface{}, loadedVars map[string]string, log log.Logger) (*latest.Config, error) {
	version, ok := data["version"].(string)
	if ok == false {
		return nil, errors.Errorf("Version is missing in devspace.yaml")
	}

	loader, ok := versionLoader[version]
	if ok == false {
		return nil, errors.Errorf("Unrecognized config version %s. Please upgrade devspace with `devspace upgrade`", version)
	}

	versionLoadFunc := loader.New

	// Load config strict
	latestConfig := versionLoadFunc()
	out, err := yaml.Marshal(data)
	if err != nil {
		return nil, err
	}
	err = yaml.UnmarshalStrict(out, latestConfig)
	if err != nil {
		return nil, errors.Errorf("Error loading config: %v", err)
	}

	// Upgrade config to latest
	for latestConfig.GetVersion() != latest.Version {
		upgradedConfig, err := latestConfig.Upgrade(log)
		if err != nil {
			return nil, errors.Errorf("Error upgrading config from version %s: %v", latestConfig.GetVersion(), err)
		}

		if loadedVars != nil {
			err = latestConfig.UpgradeVarPaths(loadedVars, log)
			if err != nil {
				return nil, errors.Errorf("Error upgrading config var paths from version %s: %v", latestConfig.GetVersion(), err)
			}
		}

		latestConfig = upgradedConfig
	}

	// Convert
	latestConfigConverted, ok := latestConfig.(*latest.Config)
	if ok == false {
		return nil, errors.Errorf("Error converting config, latest config is not the latest version")
	}

	// Update version to latest
	latestConfigConverted.Version = latest.Version

	return latestConfigConverted, nil
}

// getVariables returns only the variables from the config
func getVariables(data map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	retMap := map[interface{}]interface{}{}
	err := util.Convert(data, &retMap)
	if err != nil {
		return nil, err
	}

	vars, ok := retMap["vars"]
	if !ok {
		return map[interface{}]interface{}{
			"version": retMap["version"],
		}, nil
	}

	return map[interface{}]interface{}{
		"version": retMap["version"],
		"vars":    vars,
	}, nil
}

// getCommands returns only the commands from the config
func getCommands(data map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	retMap := map[interface{}]interface{}{}
	err := util.Convert(data, &retMap)
	if err != nil {
		return nil, err
	}

	commands, ok := retMap["commands"]
	if !ok {
		return map[interface{}]interface{}{
			"version": retMap["version"],
		}, nil
	}

	return map[interface{}]interface{}{
		"version":  retMap["version"],
		"commands": commands,
	}, nil
}

// getProfile loads a certain profile
func getProfile(data map[interface{}]interface{}, profile string) (map[interface{}]interface{}, error) {
	// Convert config
	retMap := map[interface{}]interface{}{}
	err := util.Convert(data, &retMap)
	if err != nil {
		return nil, err
	}

	// Check if a profile is defined
	if profile == "" {
		return nil, nil
	}

	// Convert to array
	profiles, ok := retMap["profiles"].([]interface{})
	if !ok {
		return nil, errors.Errorf("Couldn't load profile '%s': no profiles found", profile)
	}

	// Search for config
	for _, profileMap := range profiles {
		configMap, ok := profileMap.(map[interface{}]interface{})
		if ok && configMap["name"] == profile {
			return configMap, nil
		}
	}

	// Couldn't find config
	return nil, errors.Errorf("Couldn't find profile '%s'", profile)
}
