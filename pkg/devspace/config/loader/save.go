package loader

import (
	"io/ioutil"
	"os"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	yaml "gopkg.in/yaml.v2"
)

// Save writes the data of a config to its yaml file
func Save(path string, config *latest.Config) error {
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
