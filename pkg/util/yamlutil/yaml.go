package yamlutil

import (
	"io/ioutil"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"
)

// Convert converts an map[interface{}] to map[string] type
func Convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = Convert(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = Convert(v)
		}
	}
	return i
}

// WriteYamlToFile formats yamlData and writes it to a file
func WriteYamlToFile(yamlData interface{}, filePath string) error {
	yamlString, err := yaml.Marshal(yamlData)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, yamlString, os.ModePerm)
}

// ReadYamlFromFile reads a yaml file
func ReadYamlFromFile(filePath string, yamlTarget interface{}) error {
	yamlFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(yamlFile, yamlTarget)
}

// ToInterfaceMap converts to yaml and back to generate map[interface{}]interface{}
func ToInterfaceMap(yamlData interface{}) (map[interface{}]interface{}, error) {
	yamlString, err := yaml.Marshal(yamlData)
	if err != nil {
		return nil, err
	}

	interfaceMap := map[interface{}]interface{}{}

	err = yaml.Unmarshal(yamlString, interfaceMap)
	if err != nil {
		return nil, err
	}

	return interfaceMap, nil
}
