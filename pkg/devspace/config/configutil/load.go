package configutil

import (
	"io/ioutil"

	v1 "github.com/covexo/devspace/pkg/devspace/config/v1"
	yaml "gopkg.in/yaml.v2"
)

func loadConfig(config *v1.Config, path string) error {
	yamlFileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict(yamlFileContent, config)
}

func LoadConfigs(configs *v1.Configs, path string) error {
	yamlFileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict(yamlFileContent, configs)
}
