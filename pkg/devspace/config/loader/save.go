package loader

import (
	"io/ioutil"
	"os"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	yaml "gopkg.in/yaml.v2"
)

// Save writes the data of a config to its yaml file
func (l *configLoader) Save(config *latest.Config) error {
	// Convert to string
	configYaml, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	// Path to save the configuration to
	err = ioutil.WriteFile(ConfigPath(l.absConfigPath), configYaml, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
