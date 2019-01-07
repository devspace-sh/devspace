package configutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	v1 "github.com/covexo/devspace/pkg/devspace/config/v1"
	yaml "gopkg.in/yaml.v2"
)

// SaveBaseConfig writes the data of a config to its yaml file
func SaveBaseConfig() error {
	// Don't save custom config files
	if ConfigPath != DefaultConfigPath || OverwriteConfigPath != DefaultOverwriteConfigPath {
		return nil
	}

	// default and overwrite values
	configToIgnore := makeConfig()

	Merge(&configToIgnore, defaultConfig)
	Merge(&configToIgnore, overwriteConfig)

	// generates config without default and overwrite values
	configMapRaw, _, err := Split(config, configRaw, configToIgnore)
	if err != nil {
		return err
	}

	savePath := ConfigPath

	// Check if we have to save to configs.yaml
	if LoadedConfig != "" {
		configs := v1.Configs{}

		// Load configs
		err = loadConfigs(&configs, DefaultConfigsPath)
		if err != nil {
			return fmt.Errorf("Error loading %s: %v", DefaultConfigsPath, err)
		}

		configDefinition := configs[LoadedConfig]

		// We have to save the config in the configs.yaml
		if configDefinition.Config.Data != nil {
			saveMap, ok := configMapRaw.(*v1.Config)
			if ok == false {
				return fmt.Errorf("Error converting config. configMap is not of type *v1.Config")
			}

			configDefinition.Config.Data = saveMap
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

	configMap, _ := configMapRaw.(map[interface{}]interface{})
	configYaml, err := yaml.Marshal(configMap)
	if err != nil {
		return err
	}

	configDir := filepath.Dir(ConfigPath)
	os.MkdirAll(configDir, os.ModePerm)

	err = ioutil.WriteFile(savePath, configYaml, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
