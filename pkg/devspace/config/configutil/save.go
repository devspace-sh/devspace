package configutil

import (
	"io/ioutil"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl/walk"
	"github.com/pkg/errors"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	yaml "gopkg.in/yaml.v2"
)

func replaceVar(path, value string) (interface{}, error) {
	oldValue, _ := LoadedVars[path]
	return oldValue, nil
}

func matchVar(path, key, value string) bool {
	_, ok := LoadedVars[path+"."+key]
	return ok
}

// RestoreVars restores the variables in the config
func RestoreVars(config *latest.Config) (*latest.Config, error) {
	configMap := make(map[interface{}]interface{})

	// Copy config
	err := util.Convert(config, &configMap)
	if err != nil {
		return nil, errors.Wrap(err, "convert cloned config")
	}

	// Restore old vars values
	if len(LoadedVars) > 0 {
		walk.Walk(configMap, matchVar, replaceVar)
	}

	// Check if config exists
	_, err = os.Stat(constants.DefaultConfigPath)
	if err == nil {
		// Shallow merge with config from file to add vars, configs etc.
		bytes, err := ioutil.ReadFile(constants.DefaultConfigPath)
		if err != nil {
			return nil, err
		}

		originalConfig := map[interface{}]interface{}{}
		err = yaml.Unmarshal(bytes, &originalConfig)
		if err != nil {
			return nil, err
		}

		// Now merge missing from original into new
		for key := range originalConfig {
			if _, ok := configMap[key]; !ok {
				configMap[key] = originalConfig[key]
			}
		}
	}

	// Cloned config
	clonedConfig := &latest.Config{}

	// Copy config
	err = util.Convert(configMap, clonedConfig)
	if err != nil {
		return nil, errors.Wrap(err, "convert cloned config")
	}

	return clonedConfig, nil
}

// SaveLoadedConfig writes the data of a config to its yaml file
func SaveLoadedConfig() error {
	if len(config.Profiles) != 0 {
		return errors.Errorf("Cannot save when a profile is applied")
	}

	// RestoreVars restores the variables in the config
	clonedConfig, err := RestoreVars(config)
	if err != nil {
		return errors.Wrap(err, "restore vars")
	}

	return SaveConfig(clonedConfig)
}

// SaveConfig saves the config to file
func SaveConfig(config *latest.Config) error {
	// Convert to string
	configYaml, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	// Path to save the configuration to
	err = ioutil.WriteFile(constants.DefaultConfigPath, configYaml, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
