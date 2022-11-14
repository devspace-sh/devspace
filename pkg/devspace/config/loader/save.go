package loader

import (
	"bytes"
	"os"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	yaml "gopkg.in/yaml.v3"
)

// Save writes the data of a config to its yaml file
func Save(path string, config *latest.Config) error {
	var buffer bytes.Buffer

	yamlEncoder := yaml.NewEncoder(&buffer)
	yamlEncoder.SetIndent(2)

	err := yamlEncoder.Encode(config)
	if err != nil {
		return err
	}

	// Path to save the configuration to
	err = os.WriteFile(path, buffer.Bytes(), os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
