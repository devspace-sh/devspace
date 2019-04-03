package configutil

import (
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/stdinutil"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configs"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl/walk"
	yaml "gopkg.in/yaml.v2"
)

// VarMatchRegex is the regex to check if a value matches the devspace var format
var VarMatchRegex = regexp.MustCompile("^(.*)(\\$\\{[^\\}]+\\})(.*)$")

// VarEnvPrefix is the prefix environment variables should have in order to use them
const VarEnvPrefix = "DEVSPACE_VAR_"

// LoadedVars holds all variables that were loaded
var LoadedVars = make(map[string]string)

func varReplaceFn(path, value string) interface{} {
	// Save old value
	LoadedVars[path] = value

	matched := VarMatchRegex.FindStringSubmatch(value)
	if len(matched) != 4 {
		return ""
	}

	value = matched[2]
	varName := strings.TrimSpace(value[2 : len(value)-1])

	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatalf("Error reading generated config: %v", err)
	}

	// Find value for variable
	varValue := ""
	if os.Getenv(VarEnvPrefix+strings.ToUpper(varName)) != "" {
		envVarValue := os.Getenv(VarEnvPrefix + strings.ToUpper(varName))
		varValue = envVarValue
	} else {
		// Get current config
		currentConfig := generatedConfig.GetActive()
		if _, ok := currentConfig.Vars[varName]; !ok {
			currentConfig.Vars[varName] = AskQuestion(&configs.Variable{
				Question: ptr.String("Please enter a value for " + varName),
			})
		}

		varValue = currentConfig.Vars[varName]

		// Save config
		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatalf("Error saving generated config: %v", err)
		}
	}

	// Add matched groups again
	varValue = matched[1] + varValue + matched[3]

	// Check if we can convert val
	if i, err := strconv.Atoi(varValue); err == nil {
		return i
	} else if b, err := strconv.ParseBool(varValue); err == nil {
		return b
	}

	return varValue
}

func varMatchFn(path, key, value string) bool {
	return VarMatchRegex.MatchString(value)
}

// AskQuestion asks the user a question depending on the variable options
func AskQuestion(variable *configs.Variable) string {
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

		if variable.Options != nil {
			params.Options = *variable.Options
		} else if variable.RegexPattern != nil {
			params.ValidationRegexPattern = *variable.RegexPattern
		}
	}

	return *stdinutil.GetFromStdin(params)
}

func loadConfigFromPath(path string) (*latest.Config, error) {
	yamlFileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	out, err := resolveVars(yamlFileContent)
	if err != nil {
		return nil, err
	}

	oldConfig := map[interface{}]interface{}{}
	err = yaml.Unmarshal(out, oldConfig)
	if err != nil {
		return nil, err
	}

	newConfig, err := versions.Parse(oldConfig)
	if err != nil {
		return nil, err
	}

	return newConfig, nil
}

func loadConfigFromInterface(m map[interface{}]interface{}) (*latest.Config, error) {
	yamlFileContent, err := yaml.Marshal(m)
	if err != nil {
		return nil, err
	}

	out, err := resolveVars(yamlFileContent)
	if err != nil {
		return nil, err
	}

	oldConfig := map[interface{}]interface{}{}
	err = yaml.Unmarshal(out, oldConfig)
	if err != nil {
		return nil, err
	}

	newConfig, err := versions.Parse(oldConfig)
	if err != nil {
		return nil, err
	}

	return newConfig, nil
}

// LoadConfigs loads all the configs from the .devspace/configs.yaml
func LoadConfigs(configs *configs.Configs, path string) error {
	yamlFileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict(yamlFileContent, configs)
}

// CustomResolveVars resolves variables with a custom replace function
func CustomResolveVars(yamlFileContent []byte, matchFn func(string, string, string) bool, replaceFn func(string, string) interface{}) ([]byte, error) {
	rawConfig := make(map[interface{}]interface{})

	err := yaml.Unmarshal(yamlFileContent, &rawConfig)
	if err != nil {
		return nil, err
	}

	walk.Walk(rawConfig, matchFn, replaceFn)

	out, err := yaml.Marshal(rawConfig)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func resolveVars(yamlFileContent []byte) ([]byte, error) {
	return CustomResolveVars(yamlFileContent, varMatchFn, varReplaceFn)
}
