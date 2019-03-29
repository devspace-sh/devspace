package configutil

import (
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

// SaveBaseConfig writes the data of a config to its yaml file
func SaveBaseConfig() error {
	out, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(DefaultConfigPath, out, os.ModePerm)

	// default and overwrite values
	/*configToIgnore := latest.New()

	// generates config without default and overwrite values
	configMapRaw, _, err := Split(config, configRaw, configToIgnore)
	if err != nil {
		return err
	}

	savePath := DefaultConfigPath

	// Convert to string
	configMap, _ := configMapRaw.(map[interface{}]interface{})
	configYaml, err := yaml.Marshal(configMap)
	if err != nil {
		return err
	}

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

	return nil*/
}
