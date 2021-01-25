package loader

import (
	"io/ioutil"
	"os"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl/walk"
	"github.com/pkg/errors"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	yaml "gopkg.in/yaml.v2"
)

func (l *configLoader) replaceVar(path, value string) (interface{}, error) {
	oldValue, _ := l.options.LoadedVars[path]
	return oldValue, nil
}

func (l *configLoader) matchVar(path, key, value string) bool {
	_, ok := l.options.LoadedVars[path+"."+key]
	return ok
}

// RestoreVars restores the variables in the config
func (l *configLoader) RestoreVars(config *latest.Config) (*latest.Config, error) {
	configMap := make(map[interface{}]interface{})

	// Copy config
	err := util.Convert(config, &configMap)
	if err != nil {
		return nil, errors.Wrap(err, "convert cloned config")
	}

	// Restore old vars values
	if len(l.options.LoadedVars) > 0 {
		walk.Walk(configMap, l.matchVar, l.replaceVar)
	}

	// Check if config exists
	path := l.ConfigPath()
	_, err = os.Stat(path)
	if err == nil {
		// Shallow merge with config from file to add vars, configs etc.
		bytes, err := ioutil.ReadFile(path)
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
		return nil, errors.Wrap(err, "convert cloned config")
	}

	return clonedConfig, nil
}

var ignoreConfigKeys = []string{"images", "deployments", "dev"}

func (l *configLoader) shouldRestoreKey(key string) bool {
	for _, ignoreKey := range ignoreConfigKeys {
		if ignoreKey == key {
			return false
		}
	}
	return true
}

// Save writes the data of a config to its yaml file
func (l *configLoader) Save(config *latest.Config) error {
	// RestoreVars restores the variables in the config
	clonedConfig, err := l.RestoreVars(config)
	if err != nil {
		return errors.Wrap(err, "restore vars")
	}

	return saveConfig(l.ConfigPath(), clonedConfig)
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
