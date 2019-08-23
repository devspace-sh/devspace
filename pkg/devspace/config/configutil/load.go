package configutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/git"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/randutil"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/devspace-cloud/devspace/pkg/util/vars"
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
	"DEVSPACE_SPACE_NAMESPACE": &predefinedVarDefinition{
		ErrorMessage: fmt.Sprintf("No space configured, but predefined var DEVSPACE_SPACE_NAMESPACE is used.\n\nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space\n- `%s` to list existing spaces", ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"), ansi.Color("devspace list spaces", "white+b")),
		Fill: func(generatedConfig *generated.Config) (*string, error) {
			if generatedConfig.CloudSpace != nil {
				if generatedConfig.CloudSpace.Namespace != "" {
					return &generatedConfig.CloudSpace.Namespace, nil
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

func getPredefinedVar(name string, generatedConfig *generated.Config) (bool, string, error) {
	if variable, ok := PredefinedVars[strings.ToUpper(name)]; ok {
		if variable.Value == nil {
			return false, "", errors.New(variable.ErrorMessage)
		}

		return true, *variable.Value, nil
	}

	// Load space domain environment variable
	if strings.HasPrefix(strings.ToUpper(name), "DEVSPACE_SPACE_DOMAIN") {
		idx, err := strconv.Atoi(name[len("DEVSPACE_SPACE_DOMAIN"):])
		if err != nil {
			return false, "", fmt.Errorf("Error parsing variable %s: %v", name, err)
		}

		if generatedConfig.CloudSpace == nil {
			return false, "", fmt.Errorf("No space configured, but predefined var %s is used.\n\nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space\n- `%s` to list existing spaces", name, ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"), ansi.Color("devspace list spaces", "white+b"))
		} else if len(generatedConfig.CloudSpace.Domains) <= idx-1 {
			return false, "", fmt.Errorf("Error loading %s: Space has %d domains but domain with number %d was requested", name, len(generatedConfig.CloudSpace.Domains), idx)
		}

		return true, generatedConfig.CloudSpace.Domains[idx-1].URL, nil
	}

	return false, "", nil
}

func varReplaceFn(path, value string, generatedConfig *generated.Config) (interface{}, error) {
	// Save old value
	LoadedVars[path] = value

	return vars.ParseString(value, func(v string) (string, error) { return resolveVar(v, generatedConfig) })
}

func resolveVar(varName string, generatedConfig *generated.Config) (string, error) {
	// Is predefined variable?
	found, value, err := getPredefinedVar(varName, generatedConfig)
	if err != nil {
		return "", err
	} else if found {
		return value, nil
	}

	// Is in generated config?
	currentConfig := generatedConfig.GetActive()
	if _, ok := currentConfig.Vars[varName]; ok {
		return currentConfig.Vars[varName], nil
	}

	// Is in environment?
	if os.Getenv(VarEnvPrefix+strings.ToUpper(varName)) != "" {
		return os.Getenv(VarEnvPrefix + strings.ToUpper(varName)), nil
	} else if os.Getenv(varName) != "" {
		return os.Getenv(varName), nil
	}

	// Ask for variable
	currentConfig.Vars[varName] = AskQuestion(&configs.Variable{
		Question: ptr.String("Please enter a value for " + varName),
	})
	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		return "", fmt.Errorf("Error saving generated config: %v", err)
	}

	return currentConfig.Vars[varName], nil
}

func varMatchFn(path, key, value string) bool {
	return vars.VarMatchRegex.MatchString(value)
}

// AskQuestion asks the user a question depending on the variable options
func AskQuestion(variable *configs.Variable) string {
	params := &survey.QuestionOptions{}

	if variable == nil {
		params.Question = "Please enter a value"
	} else {
		if variable.Question == nil {
			if variable.Name == nil {
				variable.Name = ptr.String("variable")
			}

			params.Question = "Please enter a value for " + *variable.Name
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

func loadConfigFromPath(path string, generatedConfig *generated.Config) (*latest.Config, error) {
	yamlFileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	out, err := resolveVars(yamlFileContent, generatedConfig)
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

func loadConfigFromInterface(m interface{}, generatedConfig *generated.Config) (*latest.Config, error) {
	yamlFileContent, err := yaml.Marshal(m)
	if err != nil {
		return nil, err
	}

	out, err := resolveVars(yamlFileContent, generatedConfig)
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

func resolveVars(yamlFileContent []byte, generatedConfig *generated.Config) ([]byte, error) {
	err := fillPredefinedVars(generatedConfig)
	if err != nil {
		return nil, err
	}

	return CustomResolveVars(yamlFileContent, varMatchFn, func(path, value string) (interface{}, error) { return varReplaceFn(path, value, generatedConfig) })
}

func fillPredefinedVars(generatedConfig *generated.Config) error {
	for varName, predefinedVariable := range PredefinedVars {
		val, err := predefinedVariable.Fill(generatedConfig)
		if err != nil {
			return errors.Wrap(err, "fill predefined var "+varName)
		}

		predefinedVariable.Value = val
	}

	return nil
}
