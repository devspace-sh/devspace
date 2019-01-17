package configutil

import (
	"io/ioutil"
	"os"
	"strconv"

	"github.com/covexo/devspace/pkg/util/stdinutil"

	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/util/log"

	v1 "github.com/covexo/devspace/pkg/devspace/config/v1"
	yaml "gopkg.in/yaml.v2"
)

func varReplaceFn(m map[interface{}]interface{}) interface{} {
	for k, v := range m {
		key := k.(string)
		value, ok := v.(string)
		if ok == false {
			return nil
		}

		if key == "fromEnv" {
			val := os.Getenv(value)

			// Check if we can convert val
			if val == "true" {
				return true
			} else if val == "false" {
				return false
			} else if i, err := strconv.Atoi(val); err == nil {
				return i
			}

			return val
		} else if key == "fromVar" {
			generatedConfig, err := generated.LoadConfig()
			if err != nil {
				log.Fatalf("Error reading generated config: %v", err)
			}

			if configVal, ok := generatedConfig.Vars[value]; ok {
				return configVal
			}

			generatedConfig.Vars[value] = AskQuestion(&v1.Variable{
				Question: String("Please enter a value for " + value),
			})

			err = generated.SaveConfig(generatedConfig)
			if err != nil {
				log.Fatalf("Error saving generated config: %v", err)
			}

			return generatedConfig.Vars[value]
		}
	}

	return nil
}

func varMatchFn(m map[interface{}]interface{}) bool {
	if len(m) != 1 {
		return false
	}

	for k, v := range m {
		key := k.(string)
		_, ok := v.(string)
		if ok == false {
			return false
		}

		if key == "fromEnv" || key == "fromVar" {
			return true
		}
	}

	return false
}

// AskQuestion asks the user a question depending on the variable options
func AskQuestion(variable *v1.Variable) interface{} {
	params := &stdinutil.GetFromStdinParams{}

	if variable == nil {
		params.Question = "Please enter a value"
	} else {
		if variable.Question == nil {
			params.Question = "Please enter a value"
		} else {
			params.Question = *variable.Question
		}

		if variable.Default != nil {
			params.DefaultValue = *variable.Default
		}
		if variable.RegexPattern != nil {
			params.ValidationRegexPattern = *variable.RegexPattern
		}
	}

	configVal := *stdinutil.GetFromStdin(params)

	// Check if we can convert configVal
	if configVal == "true" {
		return true
	} else if configVal == "false" {
		return false
	} else if i, err := strconv.Atoi(configVal); err == nil {
		return i
	}

	return configVal
}

func loadConfigFromPath(config *v1.Config, path string) error {
	yamlFileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	out, err := resolveVars(yamlFileContent)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict(out, config)
}

func loadConfigFromInterface(config *v1.Config, m map[interface{}]interface{}) error {
	yamlFileContent, err := yaml.Marshal(m)
	if err != nil {
		return err
	}

	out, err := resolveVars(yamlFileContent)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict(out, config)
}

// LoadConfigs loads all the configs from the .devspace/configs.yaml
func LoadConfigs(configs *v1.Configs, path string) error {
	yamlFileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict(yamlFileContent, configs)
}

func resolveVars(yamlFileContent []byte) ([]byte, error) {
	rawConfig := make(map[interface{}]interface{})

	err := yaml.Unmarshal(yamlFileContent, &rawConfig)
	if err != nil {
		return nil, err
	}

	Walk(rawConfig, varMatchFn, varReplaceFn)

	out, err := yaml.Marshal(rawConfig)
	if err != nil {
		return nil, err
	}

	return out, nil
}
