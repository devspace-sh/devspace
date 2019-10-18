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

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type loader struct {
	New       config.New
	Variables config.Variables
	Commands  config.Commands
	Profile   config.Profile
}

var versionLoader = map[string]*loader{
	v1alpha1.Version: &loader{New: v1alpha1.New},
	v1alpha2.Version: &loader{New: v1alpha2.New},
	v1alpha3.Version: &loader{New: v1alpha3.New},
	v1alpha4.Version: &loader{New: v1alpha4.New},
	v1beta1.Version:  &loader{New: v1beta1.New},
	v1beta2.Version:  &loader{New: v1beta2.New},
	v1beta3.Version:  &loader{New: v1beta3.New, Variables: v1beta3.Variables, Commands: v1beta3.Commands, Profile: v1beta3.Profile},
	latest.Version:   &loader{New: latest.New, Variables: latest.Variables, Commands: latest.Commands, Profile: latest.Profile},
}

// ParseProfile loads the base config & a certain profile
func ParseProfile(data map[interface{}]interface{}, profile string) (map[interface{}]interface{}, error) {
	version, ok := data["version"].(string)
	if ok == false {
		return nil, errors.Errorf("Version is missing in devspace.yaml")
	}

	loader, ok := versionLoader[version]
	if ok == false {
		return nil, errors.Errorf("Unrecognized config version %s. Please upgrade devspace with `devspace upgrade`", version)
	}

	prepareFunc := loader.Profile
	if prepareFunc == nil {
		cloned := map[interface{}]interface{}{}
		err := util.Convert(data, &cloned)
		if err != nil {
			return nil, err
		}

		return cloned, nil
	}

	return prepareFunc(data, profile)
}

// ParseCommands parses only the commands from the config
func ParseCommands(data map[interface{}]interface{}, loadedVars map[string]string) (*latest.Config, error) {
	version, ok := data["version"].(string)
	if ok == false {
		return nil, errors.Errorf("Version is missing in devspace.yaml")
	}

	loader, ok := versionLoader[version]
	if ok == false {
		return nil, errors.Errorf("Unrecognized config version %s. Please upgrade devspace with `devspace upgrade`", version)
	}

	loadFunc := loader.Commands
	if loadFunc == nil {
		return nil, nil
	}

	strippedData, err := loadFunc(data)
	if err != nil {
		return nil, errors.Wrap(err, "loading variables")
	}

	config, err := Parse(strippedData, loadedVars)
	if err != nil {
		return nil, errors.Wrap(err, "loading vars")
	}

	return config, nil
}

// ParseVariables parses only the variables from the config
func ParseVariables(data map[interface{}]interface{}, loadedVars map[string]string) ([]*latest.Variable, error) {
	version, ok := data["version"].(string)
	if ok == false {
		// This is needed because overrides usually don't have versions
		data["version"] = latest.Version
		version = latest.Version
	}

	loader, ok := versionLoader[version]
	if ok == false {
		return nil, errors.Errorf("Unrecognized config version %s. Please upgrade devspace with `devspace upgrade`", version)
	}

	variablesLoadFunc := loader.Variables
	if variablesLoadFunc == nil {
		return []*latest.Variable{}, nil
	}

	strippedData, err := variablesLoadFunc(data)
	if err != nil {
		return nil, errors.Wrap(err, "loading variables")
	}

	config, err := Parse(strippedData, loadedVars)
	if err != nil {
		return nil, errors.Wrap(err, "loading vars")
	}

	return config.Vars, nil
}

// Parse parses the data into the latest config
func Parse(data map[interface{}]interface{}, loadedVars map[string]string) (*latest.Config, error) {
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
		upgradedConfig, err := latestConfig.Upgrade()
		if err != nil {
			return nil, errors.Errorf("Error upgrading config from version %s: %v", latestConfig.GetVersion(), err)
		}

		if loadedVars != nil {
			err = latestConfig.UpgradeVarPaths(loadedVars)
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
