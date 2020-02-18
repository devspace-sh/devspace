package loader

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudtoken "github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/util/git"
	"github.com/devspace-cloud/devspace/pkg/util/randutil"
	"github.com/mgutz/ansi"
)

// predefinedVars holds all predefined variables that can be used in the config
var predefinedVars = map[string]func(loader *configLoader) (string, error){
	"DEVSPACE_RANDOM": func(loader *configLoader) (string, error) {
		ret, err := randutil.GenerateRandomString(6)
		if err != nil {
			return "", err
		}

		return ret, nil
	},
	"DEVSPACE_TIMESTAMP": func(loader *configLoader) (string, error) {
		return strconv.FormatInt(time.Now().Unix(), 10), nil
	},
	"DEVSPACE_GIT_COMMIT": func(loader *configLoader) (string, error) {
		gitRepo := git.NewGitRepository(filepath.Dir(loader.ConfigPath()), "")
		hash, err := gitRepo.GetHash()
		if err != nil {
			return "", fmt.Errorf("No git repository found (%v), but predefined var DEVSPACE_GIT_COMMIT is used", err)
		}

		return hash[:8], nil
	},
	"DEVSPACE_SPACE": func(configLoader *configLoader) (string, error) {
		retError := fmt.Errorf("Current context is not a space, but predefined var DEVSPACE_SPACE is used.\n\nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space\n- `%s` to list existing spaces", ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"), ansi.Color("devspace list spaces", "white+b"))
		kubeLoader := configLoader.kubeConfigLoader
		options := configLoader.options
		kubeContext, err := kubeLoader.GetCurrentContext()
		if err != nil {
			return "", retError
		}
		if options.KubeContext != "" {
			kubeContext = options.KubeContext
		}

		isSpace, err := kubeLoader.IsCloudSpace(kubeContext)
		if err != nil || !isSpace {
			return "", retError
		}

		spaceID, providerName, err := kubeLoader.GetSpaceID(kubeContext)
		if err != nil {
			return "", err
		}

		loader := cloudconfig.NewLoader()
		cloudConfigData, err := loader.Load()
		if err != nil {
			return "", retError
		}

		provider := cloudconfig.GetProvider(cloudConfigData, providerName)
		if provider == nil {
			return "", retError
		}
		if provider.Spaces == nil {
			return "", retError
		}
		if provider.Spaces[spaceID] == nil {
			return "", retError
		}

		return provider.Spaces[spaceID].Space.Name, nil
	},
	"DEVSPACE_SPACE_NAMESPACE": func(configLoader *configLoader) (string, error) {
		retErr := fmt.Errorf("Current context is not a space, but predefined var DEVSPACE_SPACE_NAMESPACE is used.\n\nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space\n- `%s` to list existing spaces", ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"), ansi.Color("devspace list spaces", "white+b"))
		kubeLoader := configLoader.kubeConfigLoader
		options := configLoader.options
		kubeContext, err := kubeLoader.GetCurrentContext()
		if err != nil {
			return "", retErr
		}
		if options.KubeContext != "" {
			kubeContext = options.KubeContext
		}

		isSpace, err := kubeLoader.IsCloudSpace(kubeContext)
		if err != nil || !isSpace {
			return "", retErr
		}

		spaceID, providerName, err := kubeLoader.GetSpaceID(kubeContext)
		if err != nil {
			return "", err
		}

		loader := cloudconfig.NewLoader()
		cloudConfigData, err := loader.Load()
		if err != nil {
			return "", retErr
		}

		provider := cloudconfig.GetProvider(cloudConfigData, providerName)
		if provider == nil {
			return "", retErr
		}
		if provider.Spaces == nil {
			return "", retErr
		}
		if provider.Spaces[spaceID] == nil {
			return "", retErr
		}

		return provider.Spaces[spaceID].ServiceAccount.Namespace, nil
	},
	"DEVSPACE_USERNAME": func(configLoader *configLoader) (string, error) {
		retErr := fmt.Errorf("You are not logged into DevSpace Cloud, but predefined var DEVSPACE_USERNAME is used.\n\nPlease run: \n- `%s` to login into devspace cloud. Alternatively you can also remove the variable ${DEVSPACE_USERNAME} from your config", ansi.Color("devspace login", "white+b"))
		kubeLoader := configLoader.kubeConfigLoader
		options := configLoader.options
		kubeContext, err := kubeLoader.GetCurrentContext()
		if err != nil {
			return "", err
		}
		if options.KubeContext != "" {
			kubeContext = options.KubeContext
		}

		loader := cloudconfig.NewLoader()
		cloudConfigData, err := loader.Load()
		if err != nil {
			return "", err
		}

		_, providerName, err := kubeLoader.GetSpaceID(kubeContext)
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
			return "", retErr
		}
		if provider.Token == "" {
			return "", retErr
		}

		accountName, err := cloudtoken.GetAccountName(provider.Token)
		if err != nil {
			return "", retErr
		}

		return accountName, nil
	},
}

func (l *configLoader) resolvePredefinedVar(name string) (bool, string, error) {
	name = strings.ToUpper(name)
	if getVar, ok := predefinedVars[name]; ok {
		if l.resolvedVars == nil {
			l.resolvedVars = map[string]string{}
		}

		val, ok := l.resolvedVars[name]
		if !ok {
			val, err := getVar(l)
			if err != nil {
				return false, "", err
			}

			l.resolvedVars[name] = val
			return true, val, nil
		}

		return true, val, nil
	}

	generatedConfig, err := l.Generated()
	if err != nil {
		return false, "", nil
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
