package loader

import (
	"io/ioutil"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer/kubectl/walk"
	"github.com/pkg/errors"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
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

// Save writes the data of a config to its yaml file
func (l *configLoader) Save(config *latest.Config) error {
	// RestoreVars restores the variables in the config
	clonedConfig, err := l.RestoreVars(config)
	if err != nil {
		return errors.Wrap(err, "restore vars")
	}

	return saveConfig(clonedConfig)
}

// saveConfig saves the config to file
func saveConfig(config *latest.Config) error {
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
