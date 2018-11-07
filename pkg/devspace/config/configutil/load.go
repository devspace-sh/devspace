package configutil

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/covexo/devspace/pkg/devspace/config/v1"
	yaml "gopkg.in/yaml.v2"
)

func loadConfig(config *v1.Config, path string) error {
	workdir, _ := os.Getwd()
	yamlFileContent, err := ioutil.ReadFile(filepath.Join(workdir, path))
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict(yamlFileContent, config)
}
