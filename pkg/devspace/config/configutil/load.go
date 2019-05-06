package configutil

import (
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/mgutz/ansi"

	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudtoken "github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
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

// PredefinedVars holds all predefined variables that can be used in the config
var PredefinedVars = map[string]*string{
	PredefinedVarUsername: nil,
	PredefinedVarSpace:    nil,
}

const (
	// PredefinedVarSpace holds the space name
	PredefinedVarSpace = "DEVSPACE_SPACE"
	// PredefinedVarUsername holds the devspace cloud username
	PredefinedVarUsername = "DEVSPACE_USERNAME"
)

func varReplaceFn(path, value string) interface{} {
	// Save old value
	LoadedVars[path] = value

	matched := VarMatchRegex.FindStringSubmatch(value)
	if len(matched) != 4 {
		return ""
	}

	value = matched[2]
	varName := strings.TrimSpace(value[2 : len(value)-1])

	// Find value for variable
	varValue := ""
	if val, ok := PredefinedVars[strings.ToUpper(varName)]; ok {
		if val == nil {
			upperVarName := strings.ToUpper(varName)
			if upperVarName == PredefinedVarSpace {
				log.Fatalf("No space configured, but predefined var %s is used.\n\nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space\n- `%s` to list existing spaces", PredefinedVarSpace, ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"), ansi.Color("devspace list spaces", "white+b"))
			} else if upperVarName == PredefinedVarUsername {
				log.Fatalf("No space configured, but predefined var %s is used.\n\nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space\n- `%s` to list existing spaces", PredefinedVarUsername, ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"), ansi.Color("devspace list spaces", "white+b"))
			}

			log.Fatalf("Try to access predefined devspace variable '%s', however the value has no value", varName)
		}

		varValue = *val
	} else if os.Getenv(VarEnvPrefix+strings.ToUpper(varName)) != "" {
		envVarValue := os.Getenv(VarEnvPrefix + strings.ToUpper(varName))
		varValue = envVarValue
	} else {
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatalf("Error reading generated config: %v", err)
		}

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
	params := &survey.QuestionOptions{}

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

	return survey.Question(params)
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

func loadConfigFromInterface(m interface{}) (*latest.Config, error) {
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

// LoadConfigs loads all the configs from devspace-configs.yaml
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
	fillPredefinedVars()
	return CustomResolveVars(yamlFileContent, varMatchFn, varReplaceFn)
}

func fillPredefinedVars() {
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatalf("Error reading generated config: %v", err)
	}

	if generatedConfig.CloudSpace != nil {
		if generatedConfig.CloudSpace.Name != "" {
			PredefinedVars[PredefinedVarSpace] = &generatedConfig.CloudSpace.Name
		}
		if generatedConfig.CloudSpace.ProviderName != "" {
			cloudConfigData, err := cloudconfig.ReadCloudsConfig()
			if err != nil {
				return
			}

			dataMap := make(map[interface{}]interface{})
			err = yaml.Unmarshal(cloudConfigData, dataMap)
			if err != nil {
				return

			}

			providerMapRaw, ok := dataMap[generatedConfig.CloudSpace.ProviderName]
			if !ok {
				return
			}

			providerMap, ok := providerMapRaw.(map[interface{}]interface{})
			if !ok {
				return
			}

			tokenRaw, ok := providerMap["token"]
			if !ok {
				return
			}

			token, ok := tokenRaw.(string)
			if !ok {
				return
			}

			accountName, err := cloudtoken.GetAccountName(token)
			if err != nil {
				return
			}

			PredefinedVars[PredefinedVarUsername] = &accountName
		}
	}
}
