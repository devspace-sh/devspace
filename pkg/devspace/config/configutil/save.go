package configutil

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configs"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	yaml "gopkg.in/yaml.v2"
)

// SaveBaseConfig writes the data of a config to its yaml file
func SaveBaseConfig() error {
	// default and overwrite values
	configToIgnore := latest.New()

	// generates config without default and overwrite values
	configMapRaw, _, err := Split(config, configRaw, configToIgnore)
	if err != nil {
		return err
	}

	// Convert to string
	configMap, _ := configMapRaw.(map[interface{}]interface{})
	configYaml, err := yaml.Marshal(configMap)
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
