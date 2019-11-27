package loader

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudtoken "github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/git"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/randutil"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
)

// PredefinedVars holds all predefined variables that can be used in the config
var PredefinedVars = map[string]*predefinedVarDefinition{
	"DEVSPACE_RANDOM": &predefinedVarDefinition{
		Fill: func(options *ConfigOptions) (*string, error) {
			ret, err := randutil.GenerateRandomString(6)
			if err != nil {
				return nil, err
			}

			return &ret, nil
		},
	},
	"DEVSPACE_TIMESTAMP": &predefinedVarDefinition{
		Fill: func(options *ConfigOptions) (*string, error) {
			return ptr.String(strconv.FormatInt(time.Now().Unix(), 10)), nil
		},
	},
	"DEVSPACE_GIT_COMMIT": &predefinedVarDefinition{
		ErrorMessage: "No git repository found, but predefined var DEVSPACE_GIT_COMMIT is used",
		Fill: func(options *ConfigOptions) (*string, error) {
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
		Fill: func(options *ConfigOptions) (*string, error) {
			kubeContext, err := kubeconfig.GetCurrentContext()
			if err != nil {
				return nil, nil
			}
			if options.KubeContext != "" {
				kubeContext = options.KubeContext
			}

			isSpace, err := kubeconfig.IsCloudSpace(kubeContext)
			if err != nil || !isSpace {
				return nil, nil
			}

			spaceID, providerName, err := kubeconfig.GetSpaceID(kubeContext)
			if err != nil {
				return nil, err
			}

			loader := cloudconfig.NewLoader()
			cloudConfigData, err := loader.Load()
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
		Fill: func(options *ConfigOptions) (*string, error) {
			kubeContext, err := kubeconfig.GetCurrentContext()
			if err != nil {
				return nil, nil
			}
			if options.KubeContext != "" {
				kubeContext = options.KubeContext
			}

			isSpace, err := kubeconfig.IsCloudSpace(kubeContext)
			if err != nil || !isSpace {
				return nil, nil
			}

			spaceID, providerName, err := kubeconfig.GetSpaceID(kubeContext)
			if err != nil {
				return nil, err
			}

			loader := cloudconfig.NewLoader()
			cloudConfigData, err := loader.Load()
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
		ErrorMessage: fmt.Sprintf("You are not logged into DevSpace Cloud, but predefined var DEVSPACE_USERNAME is used.\n\nPlease run: \n- `%s` to login into devspace cloud. Alternatively you can also remove the variable ${DEVSPACE_USERNAME} from your config", ansi.Color("devspace login", "white+b")),
		Fill: func(options *ConfigOptions) (*string, error) {
			kubeContext, err := kubeconfig.GetCurrentContext()
			if err != nil {
				return nil, err
			}
			if options.KubeContext != "" {
				kubeContext = options.KubeContext
			}

			loader := cloudconfig.NewLoader()
			cloudConfigData, err := loader.Load()
			if err != nil {
				return nil, err
			}

			_, providerName, err := kubeconfig.GetSpaceID(kubeContext)
			if err != nil {
				// use global provider config as fallback
				if cloudConfigData.Default != "" {
					providerName = cloudConfigData.Default
				} else {
					providerName = cloudconfig.DevSpaceCloudProviderName
				}
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
	Fill         func(*ConfigOptions) (*string, error)
}

func fillPredefinedVars(options *ConfigOptions) error {
	for varName, predefinedVariable := range PredefinedVars {
		val, err := predefinedVariable.Fill(options)
		if err != nil {
			return errors.Wrap(err, "fill predefined var "+varName)
		}

		predefinedVariable.Value = val
	}

	return nil
}

func getPredefinedVar(name string, generatedConfig *generated.Config, options *ConfigOptions) (bool, string, error) {
	name = strings.ToUpper(name)
	if variable, ok := PredefinedVars[name]; ok {
		if variable.Value == nil {
			return false, "", errors.New(variable.ErrorMessage)
		}

		return true, *variable.Value, nil
	}

	// Load space domain environment variable
	if strings.HasPrefix(name, "DEVSPACE_SPACE_DOMAIN") {
		// Check if its in generated config
		if val, ok := generatedConfig.Vars[name]; ok {
			return true, val, nil
		}

		return true, name, nil
	}

	return false, "", nil
}
