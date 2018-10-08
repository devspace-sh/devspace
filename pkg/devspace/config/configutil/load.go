package configutil

import (
	"io/ioutil"

	"github.com/covexo/devspace/pkg/devspace/config/v1"
	yaml "gopkg.in/yaml.v2"
)

func loadConfig(config *v1.Config, path string) error {
	yamlFileContent, err := ioutil.ReadFile(workdir + path)

	if err != nil {
		return err
	}
	return yaml.Unmarshal(yamlFileContent, config)
}
