package yamlutil

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func UnmarshalStrictJSON(data []byte, out interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(out)
}

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
		// print the config with lines
		lines := strings.Split(string(data), "\n")
		extraLines := []string{"Parsed Config:"}
		for i, v := range lines {
			if v == "" {
				continue
			}
			extraLines = append(extraLines, fmt.Sprintf("  %d: %s", i+1, v))
		}
		extraLines = append(extraLines, "Errors:")

		for i := range typeError.Errors {
			typeError.Errors[i] = strings.ReplaceAll(typeError.Errors[i], "!!seq", "an array")
			typeError.Errors[i] = strings.ReplaceAll(typeError.Errors[i], "!!str", "string")
			typeError.Errors[i] = strings.ReplaceAll(typeError.Errors[i], "!!map", "an object")
			typeError.Errors[i] = strings.ReplaceAll(typeError.Errors[i], "!!int", "number")
			typeError.Errors[i] = strings.ReplaceAll(typeError.Errors[i], "!!bool", "boolean")

			// add line to error
			match := lineRegEx.FindSubmatch([]byte(typeError.Errors[i]))
			if len(match) > 1 {
				line, lineErr := strconv.Atoi(string(match[1]))
				if lineErr == nil {
					line = line - 1
					lines := strings.Split(string(data), "\n")
					if line < len(lines) {
						typeError.Errors[i] = "  " + typeError.Errors[i] + fmt.Sprintf(" (line %d: %s)", line+1, strings.TrimSpace(lines[line]))
					}
				}
			}
		}

		extraLines = append(extraLines, typeError.Errors...)
		typeError.Errors = extraLines
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
	return os.WriteFile(filePath, yamlString, os.ModePerm)
}

// ReadYamlFromFile reads a yaml file
func ReadYamlFromFile(filePath string, yamlTarget interface{}) error {
	yamlFile, err := os.ReadFile(filePath)
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
