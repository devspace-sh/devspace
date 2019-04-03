package configutil

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl/walk"
	"github.com/pkg/errors"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configs"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	yaml "gopkg.in/yaml.v2"
)

func replaceVar(path, value string) interface{} {
	oldValue, _ := LoadedVars[path]
	return oldValue
}

func matchVar(path, key, value string) bool {
	_, ok := LoadedVars[path+"."+key]
	return ok
}

// SaveBaseConfig writes the data of a config to its yaml file
func SaveBaseConfig() error {
	// Convert to map[interface{}]interface{}
	configMap := make(map[interface{}]interface{})

	// Copy config
	err := util.Convert(config, &configMap)
	if err != nil {
		return errors.Wrap(err, "convert config map to map interface")
	}

	// Restore old vars values
	if len(LoadedVars) >= 1 {
		walk.Walk(configMap, matchVar, replaceVar)
	}

	// Cloned config
	clonedConfig := latest.Config{}

	// Copy config
	err = util.Convert(configMap, &clonedConfig)
	if err != nil {
		return errors.Wrap(err, "convert cloned config")
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

	// Convert to string
	configYaml, err := yaml.Marshal(clonedConfig)
	if err != nil {
		return err
	}

	// Path to save the configuration to
	savePath := DefaultConfigPath

	// Check if we have to save to configs.yaml
	if LoadedConfig != "" {
		configs := configs.Configs{}

		// Load configs
		err = LoadConfigs(&configs, DefaultConfigsPath)
		if err != nil {
			return fmt.Errorf("Error loading %s: %v", DefaultConfigsPath, err)
		}

		configDefinition := configs[LoadedConfig]

		// We have to save the config in the configs.yaml
		if configDefinition.Config.Data != nil {
			configDefinition.Config.Data = configMap
			configYaml, err := yaml.Marshal(configs)
			if err != nil {
				return err
			}

			err = ioutil.WriteFile(DefaultConfigsPath, configYaml, os.ModePerm)
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
