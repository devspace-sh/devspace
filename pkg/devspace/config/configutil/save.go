package configutil

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl/walk"
	"github.com/pkg/errors"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configs"
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

	// Cloned config
	clonedConfig := &latest.Config{}

	// Copy config
	err = util.Convert(configMap, clonedConfig)
	if err != nil {
		return nil, errors.Wrap(err, "convert cloned config")
	}

	// Erase default values
	if clonedConfig.Dev != nil && *clonedConfig.Dev == (latest.DevConfig{}) {
		clonedConfig.Dev = nil
	}
	if clonedConfig.Cluster != nil && *clonedConfig.Cluster == (latest.Cluster{}) {
		clonedConfig.Cluster = nil
	}
	if clonedConfig.Deployments != nil && len(*clonedConfig.Deployments) == 0 {
		clonedConfig.Deployments = nil
	}
	if clonedConfig.Images != nil && len(*clonedConfig.Images) == 0 {
		clonedConfig.Images = nil
	}

	return clonedConfig, nil
}

// SaveLoadedConfig writes the data of a config to its yaml file
func SaveLoadedConfig() error {
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
	savePath := constants.DefaultConfigPath

	// Check if we have to save to configs.yaml
	if LoadedConfig != "" {
		configs := configs.Configs{}

		// Load configs
		err = LoadConfigs(&configs, constants.DefaultConfigsPath)
		if err != nil {
			return fmt.Errorf("Error loading %s: %v", constants.DefaultConfigsPath, err)
		}

		configDefinition := configs[LoadedConfig]

		// We have to save the config in the configs.yaml
		if configDefinition.Config.Data != nil {
			configDefinition.Config.Data = config
			configYaml, err := yaml.Marshal(configs)
			if err != nil {
				return err
			}

			err = ioutil.WriteFile(constants.DefaultConfigsPath, configYaml, os.ModePerm)
			if err != nil {
				return err
			}

			return nil
		}

		// Save config in save path
		savePath = *configDefinition.Config.Path
	}

	err = ioutil.WriteFile(savePath, configYaml, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
