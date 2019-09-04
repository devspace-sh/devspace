package configutil

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/git"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/randutil"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/devspace-cloud/devspace/pkg/util/vars"

	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudtoken "github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configs"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl/walk"
	"github.com/mgutz/ansi"
	yaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
)

// VarEnvPrefix is the prefix environment variables should have in order to use them
const VarEnvPrefix = "DEVSPACE_VAR_"

// LoadedVars holds all variables that were loaded
var LoadedVars = make(map[string]string)

// PredefinedVars holds all predefined variables that can be used in the config
var PredefinedVars = map[string]*predefinedVarDefinition{
	"DEVSPACE_RANDOM": &predefinedVarDefinition{
		Fill: func(ctx context.Context) (*string, error) {
			ret, err := randutil.GenerateRandomString(6)
			if err != nil {
				return nil, err
			}

			return &ret, nil
		},
	},
	"DEVSPACE_TIMESTAMP": &predefinedVarDefinition{
		Fill: func(ctx context.Context) (*string, error) {
			return ptr.String(strconv.FormatInt(time.Now().Unix(), 10)), nil
		},
	},
	"DEVSPACE_GIT_COMMIT": &predefinedVarDefinition{
		ErrorMessage: "No git repository found, but predefined var DEVSPACE_GIT_COMMIT is used",
		Fill: func(ctx context.Context) (*string, error) {
			gitRepo := git.NewGitRepository(".", "")

			hash, err := gitRepo.GetHash()
			if err != nil {
				return nil, nil
			}

			return ptr.String(hash[:8]), nil
		},
	},
	"DEVSPACE_SPACE": &predefinedVarDefinition{
		ErrorMessage: fmt.Sprintf("Current context is not a space, but predefined var DEVSPACE_SPACE is used.\n\nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space\n- `%s` to list existing spaces", ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"), ansi.Color("devspace list spaces", "white+b")),
		Fill: func(ctx context.Context) (*string, error) {
			kubeContext, err := kubeconfig.GetCurrentContext()
			if err != nil {
				return nil, nil
			}
			if ctx.Value(constants.KubeContextKey) != nil {
				kubeContext = ctx.Value(constants.KubeContextKey).(string)
			}

			isSpace, err := kubeconfig.IsCloudSpace(kubeContext)
			if err != nil || !isSpace {
				return nil, nil
			}

			spaceID, providerName, err := kubeconfig.GetSpaceID(kubeContext)
			if err != nil {
				return nil, err
			}

			cloudConfigData, err := cloudconfig.ParseProviderConfig()
			if err != nil {
				return nil, nil
			}

			provider := cloudconfig.GetProvider(cloudConfigData, providerName)
			if provider == nil {
				return nil, nil
			}
			if provider.Spaces == nil {
				return nil, nil
			}
			if provider.Spaces[spaceID] == nil {
				return nil, nil
			}

			return &provider.Spaces[spaceID].Space.Name, nil
		},
	},
	"DEVSPACE_SPACE_NAMESPACE": &predefinedVarDefinition{
		ErrorMessage: fmt.Sprintf("Current context is not a space, but predefined var DEVSPACE_SPACE_NAMESPACE is used.\n\nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space\n- `%s` to list existing spaces", ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"), ansi.Color("devspace list spaces", "white+b")),
		Fill: func(ctx context.Context) (*string, error) {
			kubeContext, err := kubeconfig.GetCurrentContext()
			if err != nil {
				return nil, nil
			}
			if ctx.Value(constants.KubeContextKey) != nil {
				kubeContext = ctx.Value(constants.KubeContextKey).(string)
			}

			isSpace, err := kubeconfig.IsCloudSpace(kubeContext)
			if err != nil || !isSpace {
				return nil, nil
			}

			spaceID, providerName, err := kubeconfig.GetSpaceID(kubeContext)
			if err != nil {
				return nil, err
			}

			cloudConfigData, err := cloudconfig.ParseProviderConfig()
			if err != nil {
				return nil, nil
			}

			provider := cloudconfig.GetProvider(cloudConfigData, providerName)
			if provider == nil {
				return nil, nil
			}
			if provider.Spaces == nil {
				return nil, nil
			}
			if provider.Spaces[spaceID] == nil {
				return nil, nil
			}

			return &provider.Spaces[spaceID].ServiceAccount.Namespace, nil
		},
	},
	"DEVSPACE_USERNAME": &predefinedVarDefinition{
		ErrorMessage: fmt.Sprintf("Current context is not a space, but predefined var DEVSPACE_USERNAME is used.\n\nPlease run: \n- `%s` to login into devspace cloud. Alternatively you can also remove the variable ${DEVSPACE_USERNAME} from your config", ansi.Color("devspace login", "white+b")),
		Fill: func(ctx context.Context) (*string, error) {
			kubeContext, err := kubeconfig.GetCurrentContext()
			if err != nil {
				return nil, nil
			}
			if ctx.Value(constants.KubeContextKey) != nil {
				kubeContext = ctx.Value(constants.KubeContextKey).(string)
			}

			_, providerName, err := kubeconfig.GetSpaceID(kubeContext)
			if err != nil {
				// use global provider config as fallback
				providerName = cloudconfig.DevSpaceCloudProviderName
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
	Fill         func(ctx context.Context) (*string, error)
}

func getPredefinedVar(ctx context.Context, name string) (bool, string, error) {
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

		kubeContext, err := kubeconfig.GetCurrentContext()
		if err != nil {
			return false, "", errors.Wrap(err, "get current context")
		}
		if ctx.Value(constants.KubeContextKey) != nil {
			kubeContext = ctx.Value(constants.KubeContextKey).(string)
		}

		spaceID, providerName, err := kubeconfig.GetSpaceID(kubeContext)
		if err != nil {
			return false, "", fmt.Errorf("No space configured, but predefined var %s is used.\n\nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space\n- `%s` to list existing spaces", name, ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"), ansi.Color("devspace list spaces", "white+b"))
		}

		cloudConfigData, err := cloudconfig.ParseProviderConfig()
		if err != nil {
			return false, "", errors.Wrap(err, "parse provider config")
		}

		provider := cloudconfig.GetProvider(cloudConfigData, providerName)
		if provider == nil {
			return false, "", fmt.Errorf("Couldn't find space provider: %s", providerName)
		}
		if provider.Spaces == nil {
			return false, "", fmt.Errorf("No space configured, but predefined var %s is used.\n\nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space\n- `%s` to list existing spaces", name, ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"), ansi.Color("devspace list spaces", "white+b"))
		}
		if provider.Spaces[spaceID] == nil {
			return false, "", fmt.Errorf("No space configured, but predefined var %s is used.\n\nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space\n- `%s` to list existing spaces", name, ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"), ansi.Color("devspace list spaces", "white+b"))
		}

		if len(provider.Spaces[spaceID].Space.Domains) <= idx-1 {
			return false, "", fmt.Errorf("Error loading %s: Space has %d domains but domain with number %d was requested", name, len(provider.Spaces[spaceID].Space.Domains), idx)
		}

		return true, provider.Spaces[spaceID].Space.Domains[idx-1].URL, nil
	}

	return false, "", nil
}

func varReplaceFn(ctx context.Context, path, value string, generatedConfig *generated.Config) (interface{}, error) {
	// Save old value
	LoadedVars[path] = value

	return vars.ParseString(value, func(v string) (string, error) { return resolveVar(ctx, v, generatedConfig) })
}

func resolveVar(ctx context.Context, varName string, generatedConfig *generated.Config) (string, error) {
	// Is predefined variable?
	found, value, err := getPredefinedVar(ctx, varName)
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

func loadConfigFromPath(ctx context.Context, path string, generatedConfig *generated.Config) (*latest.Config, error) {
	yamlFileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	out, err := resolveVars(ctx, yamlFileContent, generatedConfig)
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

func loadConfigFromInterface(ctx context.Context, m interface{}, generatedConfig *generated.Config) (*latest.Config, error) {
	yamlFileContent, err := yaml.Marshal(m)
	if err != nil {
		return nil, err
	}

	out, err := resolveVars(ctx, yamlFileContent, generatedConfig)
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

func resolveVars(ctx context.Context, yamlFileContent []byte, generatedConfig *generated.Config) ([]byte, error) {
	err := fillPredefinedVars(ctx)
	if err != nil {
		return nil, err
	}

	return CustomResolveVars(yamlFileContent, varMatchFn, func(path, value string) (interface{}, error) { return varReplaceFn(ctx, path, value, generatedConfig) })
}

func fillPredefinedVars(ctx context.Context) error {
	for varName, predefinedVariable := range PredefinedVars {
		val, err := predefinedVariable.Fill(ctx)
		if err != nil {
			return errors.Wrap(err, "fill predefined var "+varName)
		}

		predefinedVariable.Value = val
	}

	return nil
}
