package configutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/git"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/randutil"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"

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
var PredefinedVars = map[string]*predefinedVarDefinition{
	"DEVSPACE_RANDOM": &predefinedVarDefinition{
		Fill: func(generatedConfig *generated.Config) (*string, error) {
			ret, err := randutil.GenerateRandomString(6)
			if err != nil {
				return nil, err
			}

			return &ret, nil
		},
	},
	"DEVSPACE_TIMESTAMP": &predefinedVarDefinition{
		Fill: func(generatedConfig *generated.Config) (*string, error) {
			return ptr.String(strconv.FormatInt(time.Now().Unix(), 10)), nil
		},
	},
	"DEVSPACE_GIT_COMMIT": &predefinedVarDefinition{
		ErrorMessage: "No git repository found, but predefined var DEVSPACE_GIT_COMMIT is used",
		Fill: func(generatedConfig *generated.Config) (*string, error) {
			gitRepo := git.NewGitRepository(".", "")

			hash, err := gitRepo.GetHash()
			if err != nil {
				return nil, nil
			}

			return ptr.String(hash[:8]), nil
		},
	},
	"DEVSPACE_SPACE": &predefinedVarDefinition{
		ErrorMessage: fmt.Sprintf("No space configured, but predefined var DEVSPACE_SPACE is used.\n\nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space\n- `%s` to list existing spaces", ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"), ansi.Color("devspace list spaces", "white+b")),
		Fill: func(generatedConfig *generated.Config) (*string, error) {
			if generatedConfig.CloudSpace != nil {
				if generatedConfig.CloudSpace.Name != "" {
					return &generatedConfig.CloudSpace.Name, nil
				}
			}

			return nil, nil
		},
	},
	"DEVSPACE_USERNAME": &predefinedVarDefinition{
		ErrorMessage: fmt.Sprintf("Not logged into Devspace Cloud, but predefined var DEVSPACE_USERNAME is used.\n\nPlease run: \n- `%s` to login into devspace cloud. Alternatively you can also remove the variable ${DEVSPACE_USERNAME} from your config", ansi.Color("devspace login", "white+b")),
		Fill: func(generatedConfig *generated.Config) (*string, error) {
			providerName := cloudconfig.DevSpaceCloudProviderName
			if generatedConfig.CloudSpace != nil {
				if generatedConfig.CloudSpace.ProviderName != "" {
					providerName = generatedConfig.CloudSpace.ProviderName
				}
			}

			cloudConfigData, err := cloudconfig.ParseProviderConfig()
			if err != nil {
				return nil, nil
			}

			provider := cloudconfig.GetProvider(cloudConfigData, providerName)
			if provider == nil {
				return nil, nil
			}
			if provider.Token == "" {
				return nil, nil
			}

			accountName, err := cloudtoken.GetAccountName(provider.Token)
			if err != nil {
				return nil, nil
			}

			return &accountName, nil
		},
	},
}

type predefinedVarDefinition struct {
	Value        *string
	ErrorMessage string
	Fill         func(generatedConfig *generated.Config) (*string, error)
}

func varReplaceFn(path, value string) (interface{}, error) {
	// Save old value
	LoadedVars[path] = value

	matched := VarMatchRegex.FindStringSubmatch(value)
	if len(matched) != 4 {
		return "", nil
	}

	value = matched[2]
	varName := strings.TrimSpace(value[2 : len(value)-1])

	// Find value for variable
	varValue := ""
	if variable, ok := PredefinedVars[strings.ToUpper(varName)]; ok {
		if variable.Value == nil {
			return nil, errors.New(variable.ErrorMessage)
		}

		varValue = *variable.Value
	} else if os.Getenv(VarEnvPrefix+strings.ToUpper(varName)) != "" {
		envVarValue := os.Getenv(VarEnvPrefix + strings.ToUpper(varName))
		varValue = envVarValue
	} else {
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("Error reading generated config: %v", err)
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
			return nil, fmt.Errorf("Error saving generated config: %v", err)
		}
	}

	// Add matched groups again
	varValue = matched[1] + varValue + matched[3]

	// Check if we can convert val
	if i, err := strconv.Atoi(varValue); err == nil {
		return i, nil
	} else if b, err := strconv.ParseBool(varValue); err == nil {
		return b, nil
	}

	return varValue, nil
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
		} else if variable.ValidationPattern != nil {
			params.ValidationRegexPattern = *variable.ValidationPattern

			if variable.ValidationMessage != nil {
				params.ValidationMessage = *variable.ValidationMessage
			}
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
func CustomResolveVars(yamlFileContent []byte, matchFn func(string, string, string) bool, replaceFn func(string, string) (interface{}, error)) ([]byte, error) {
	rawConfig := make(map[interface{}]interface{})

	err := yaml.Unmarshal(yamlFileContent, &rawConfig)
	if err != nil {
		return nil, err
	}

	err = walk.Walk(rawConfig, matchFn, replaceFn)
	if err != nil {
		return nil, err
	}

	out, err := yaml.Marshal(rawConfig)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func resolveVars(yamlFileContent []byte) ([]byte, error) {
	err := fillPredefinedVars()
	if err != nil {
		return nil, err
	}

	return CustomResolveVars(yamlFileContent, varMatchFn, varReplaceFn)
}

func fillPredefinedVars() error {
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		return fmt.Errorf("Error reading generated config: %v", err)
	}

	for varName, predefinedVariable := range PredefinedVars {
		val, err := predefinedVariable.Fill(generatedConfig)
		if err != nil {
			return errors.Wrap(err, "fill predefined var "+varName)
		}

		predefinedVariable.Value = val
	}

	return nil
}
