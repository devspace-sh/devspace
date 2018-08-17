package yamlutil

import (
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

func WriteYamlToFile(yamlData interface{}, filePath string) error {
	yamlString, yamlErr := yaml.Marshal(yamlData)

	if yamlErr != nil {
		return yamlErr
	}
	return ioutil.WriteFile(filePath, yamlString, os.ModePerm)
}

func ReadYamlFromFile(filePath string, yamlTarget interface{}) error {
	yamlFile, err := ioutil.ReadFile(filePath)

	if err != nil {
		return err
	}
	return yaml.Unmarshal(yamlFile, yamlTarget)
}
