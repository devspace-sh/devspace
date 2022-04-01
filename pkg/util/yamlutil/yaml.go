package yamlutil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// UnmarshalString decodes the given string into an object and returns a prettified string
func UnmarshalString(data string, out interface{}) error {
	return Unmarshal([]byte(data), out)
}

var lineRegEx = regexp.MustCompile(`^line ([0-9]+):`)

func UnmarshalStrict(data []byte, out interface{}) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	err := decoder.Decode(out)
	return prettifyError(data, err)
}

// Unmarshal decodes the given byte into an object and returns a prettified string
func Unmarshal(data []byte, out interface{}) error {
	err := yaml.Unmarshal(data, out)
	return prettifyError(data, err)
}

func prettifyError(data []byte, err error) error {
	// check if type error
	if typeError, ok := err.(*yaml.TypeError); ok {
		for i := range typeError.Errors {
			typeError.Errors[i] = strings.Replace(typeError.Errors[i], "!!seq", "an array", -1)
			typeError.Errors[i] = strings.Replace(typeError.Errors[i], "!!str", "string", -1)
			typeError.Errors[i] = strings.Replace(typeError.Errors[i], "!!map", "an object", -1)
			typeError.Errors[i] = strings.Replace(typeError.Errors[i], "!!int", "number", -1)
			typeError.Errors[i] = strings.Replace(typeError.Errors[i], "!!bool", "boolean", -1)

			// add line to error
			match := lineRegEx.FindSubmatch([]byte(typeError.Errors[i]))
			if len(match) > 1 {
				line, lineErr := strconv.Atoi(string(match[1]))
				if lineErr == nil {
					line = line - 1
					lines := strings.Split(string(data), "\n")
					if line < len(lines) {
						typeError.Errors[i] += fmt.Sprintf(" (line %d: %s)", line+1, strings.TrimSpace(lines[line]))
					}
				}
			}
		}
	}

	return err
}

// Convert converts an map[interface{}] to map[string] type
func Convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[string]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k] = Convert(v)
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

// ToInterfaceMap converts to yaml and back to generate map[string]interface{}
func ToInterfaceMap(yamlData interface{}) (map[string]interface{}, error) {
	yamlString, err := yaml.Marshal(yamlData)
	if err != nil {
		return nil, err
	}

	interfaceMap := map[string]interface{}{}

	err = yaml.Unmarshal(yamlString, interfaceMap)
	if err != nil {
		return nil, err
	}

	return interfaceMap, nil
}
