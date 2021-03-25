package loader

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"io/ioutil"
	"os"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	"github.com/pkg/errors"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	yaml "gopkg.in/yaml.v2"
)

var ignoreConfigKeys = []string{"images", "deployments", "dev"}

func (l *configLoader) shouldRestoreKey(key string) bool {
	for _, ignoreKey := range ignoreConfigKeys {
		if ignoreKey == key {
			return false
		}
	}
	return true
}

// SaveGenerated is a convenience method to save the generated config
func (l *configLoader) SaveGenerated(generatedConfig *generated.Config) error {
	return generated.NewConfigLoader("").Save(generatedConfig)
}

// Save writes the data of a config to its yaml file
func (l *configLoader) Save(config *latest.Config) error {
	configMap := make(map[interface{}]interface{})

	// Copy config
	err := util.Convert(config, &configMap)
	if err != nil {
		return errors.Wrap(err, "convert cloned config")
	}

	// Check if config exists
	path := ConfigPath(l.configPath)
	_, err = os.Stat(path)
	if err == nil {
		// Shallow merge with config from file to add vars, configs etc.
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		originalConfig := map[interface{}]interface{}{}
		err = yaml.Unmarshal(bytes, &originalConfig)
		if err != nil {
			return err
		}

		// Now merge missing from original into new
		for key := range originalConfig {
			keyString, isString := key.(string)
			if !isString {
				continue
			}

			if _, ok := configMap[keyString]; !ok && l.shouldRestoreKey(keyString) {
				configMap[keyString] = originalConfig[keyString]
			}
		}
	}

	// Cloned config
	clonedConfig := &latest.Config{}

	// Copy config
	err = util.Convert(configMap, clonedConfig)
	if err != nil {
		return errors.Wrap(err, "convert cloned config")
	}

	return saveConfig(ConfigPath(l.configPath), clonedConfig)
}

// saveConfig saves the config to file
func saveConfig(path string, config *latest.Config) error {
	// Convert to string
	configYaml, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	// Path to save the configuration to
	err = ioutil.WriteFile(path, configYaml, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
