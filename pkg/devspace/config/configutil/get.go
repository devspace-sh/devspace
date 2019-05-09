package configutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	yaml "gopkg.in/yaml.v2"
	"k8s.io/client-go/tools/clientcmd"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"

	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configs"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/mgutz/ansi"
)

// ConfigInterface defines the pattern of every config
type ConfigInterface interface{}

// DefaultConfigsPath is the default configs path to use
const DefaultConfigsPath = "devspace-configs.yaml"

// DefaultConfigPath is the default config path to use
const DefaultConfigPath = "devspace.yaml"

// DefaultVarsPath is the default vars path to use
const DefaultVarsPath = "devspace-vars.yaml"

// DefaultDevSpaceSelectorName is the default app selector
const DefaultDevSpaceSelectorName = "app-selector"

// LoadedConfig is the config that was loaded from the configs file
var LoadedConfig string

// Global config vars
var config *latest.Config // merged config

// Thread-safety helper
var getConfigOnce sync.Once
var validateOnce sync.Once

// ConfigExists checks whether the yaml file for the config exists or the configs.yaml exists
func ConfigExists() bool {
	return configExistsInPath(".")
}

// configExistsInPath checks wheter a devspace configuration exists at a certain path
func configExistsInPath(path string) bool {
	// Needed for testing
	if config != nil {
		return true
	}

	// Check devspace.yaml
	_, err := os.Stat(filepath.Join(path, DefaultConfigPath))
	if err == nil {
		return true
	}

	// Check devspace-configs.yaml
	_, err = os.Stat(filepath.Join(path, DefaultConfigsPath))
	if err == nil {
		return true
	}

	// Check old .devspace/config.yaml
	_, err = os.Stat(filepath.Join(path, ".devspace", "config.yaml"))
	if err == nil {
		return true
	}

	// Check old .devspace/configs.yaml
	_, err = os.Stat(filepath.Join(path, ".devspace", "configs.yaml"))
	if err == nil {
		return true
	}

	return false // Normal config file found
}

// InitConfig initializes the config objects
func InitConfig() *latest.Config {
	getConfigOnce.Do(func() {
		config = latest.New().(*latest.Config)
	})

	return config
}

// GetBaseConfig returns the config unmerged with potential overwrites
func GetBaseConfig() *latest.Config {
	GetConfigWithoutDefaults(false)
	ValidateOnce()

	return config
}

// GetConfig returns the config merged with all potential overwrite files
func GetConfig() *latest.Config {
	GetConfigWithoutDefaults(true)
	ValidateOnce()

	return config
}

func loadBaseConfigFromPath(basePath string, loadConfig string, cache *generated.CacheConfig) (*latest.Config, *configs.ConfigDefinition, error) {
	var (
		config           = latest.New().(*latest.Config)
		configRaw        = latest.New().(*latest.Config)
		configDefinition *configs.ConfigDefinition
		configPath       = filepath.Join(basePath, DefaultConfigPath)
		configsPath      = filepath.Join(basePath, DefaultConfigsPath)
		varsPath         = filepath.Join(basePath, DefaultVarsPath)
	)

	// Check if configs.yaml exists
	_, err := os.Stat(configsPath)
	if err == nil {
		configs := configs.Configs{}

		// Get configs
		err = LoadConfigs(&configs, configsPath)
		if err != nil {
			return nil, nil, fmt.Errorf("Error loading %s: %v", configsPath, err)
		}

		// Check if active config exists
		if _, ok := configs[loadConfig]; ok == false {
			return nil, nil, fmt.Errorf("Config %s couldn't be found", loadConfig)
		}

		// Get real config definition
		configDefinition = configs[loadConfig]
		if configDefinition.Config == nil {
			return nil, nil, fmt.Errorf("Config %s couldn't be found", loadConfig)
		}

		// Ask questions
		if configDefinition.Vars != nil {
			vars, err := loadVarsFromWrapper(basePath, configDefinition.Vars)
			if err != nil {
				return nil, nil, fmt.Errorf("Error loading vars: %v", err)
			}

			err = askQuestions(cache, vars)
			if err != nil {
				return nil, nil, fmt.Errorf("Error filling vars: %v", err)
			}
		}

		// Load config
		configRaw, err = loadConfigFromWrapper(basePath, configDefinition.Config)
		if err != nil {
			return nil, nil, err
		}
	} else {
		_, err := os.Stat(varsPath)
		if err == nil {
			vars := []*configs.Variable{}
			yamlFileContent, err := ioutil.ReadFile(varsPath)
			if err != nil {
				return nil, nil, fmt.Errorf("Error loading %s: %v", varsPath, err)
			}

			err = yaml.UnmarshalStrict(yamlFileContent, vars)
			if err != nil {
				return nil, nil, fmt.Errorf("Error parsing %s: %v", varsPath, err)
			}

			// Ask questions
			err = askQuestions(cache, vars)
			if err != nil {
				return nil, nil, fmt.Errorf("Error filling vars: %v", err)
			}
		}

		configRaw, err = loadConfigFromPath(configPath)
		if err != nil {
			return nil, nil, fmt.Errorf("Loading config: %v", err)
		}
	}

	Merge(&config, deepCopy(configRaw))
	return config, configDefinition, nil
}

// GetConfigFromPath loads the config from a given base path
func GetConfigFromPath(basePath string, loadConfig string, cache *generated.CacheConfig) (*latest.Config, error) {
	config, configDefinition, err := loadBaseConfigFromPath(basePath, loadConfig, cache)
	if err != nil {
		return nil, err
	}

	// Check if we should load overrides
	if configDefinition != nil {
		if configDefinition.Overrides != nil {
			for index, configWrapper := range *configDefinition.Overrides {
				overwriteConfig, err := loadConfigFromWrapper(basePath, configWrapper)
				if err != nil {
					return nil, fmt.Errorf("Error loading override config at index %d: %v", index, err)
				}

				Merge(&config, overwriteConfig)
			}
		}
	}

	return config, nil
}

// GetConfigWithoutDefaults returns the config without setting the default values
func GetConfigWithoutDefaults(loadOverwrites bool) *latest.Config {
	getConfigOnce.Do(func() {
		var (
			err              error
			configDefinition *configs.ConfigDefinition
		)

		// Get generated config
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Panicf("Error loading %s: %v", generated.ConfigPath, err)
		}

		// Get config to load
		LoadedConfig = generatedConfig.ActiveConfig

		// Load base config
		config, configDefinition, err = loadBaseConfigFromPath(".", LoadedConfig, generatedConfig.GetActive())
		if err != nil {
			log.Fatal(err)
		}

		// Reset loaded config if there was no configs.yaml
		if configDefinition == nil {
			LoadedConfig = ""
		}

		// Check if we should load overrides
		if loadOverwrites {
			if configDefinition != nil {
				if configDefinition.Overrides != nil {
					for index, configWrapper := range *configDefinition.Overrides {
						overwriteConfig, err := loadConfigFromWrapper(".", configWrapper)
						if err != nil {
							log.Fatalf("Error loading override config at index %d: %v", index, err)
						}

						Merge(&config, overwriteConfig)
					}

					log.Infof("Loaded config %s from %s with %d overrides", LoadedConfig, DefaultConfigsPath, len(*configDefinition.Overrides))
				} else {
					log.Infof("Loaded config %s from %s", LoadedConfig, DefaultConfigsPath)
				}
			} else {
				log.Infof("Loaded config from %s", DefaultConfigPath)
			}

			// Exchange kube context if necessary, but only if we don't load the base config
			// we do this to avoid saving the kube context on commands like
			// devspace add deployment && devspace add image etc.
			if generatedConfig.CloudSpace != nil {
				if config.Cluster == nil || (config.Cluster.KubeContext == nil && config.Cluster.APIServer == nil) {
					if generatedConfig.CloudSpace.KubeContext == "" {
						log.Fatalf("No space configured\n\nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space\n- `%s` to list existing spaces", ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"), ansi.Color("devspace list spaces", "white+b"))
					}

					config.Cluster = &latest.Cluster{
						KubeContext: &generatedConfig.CloudSpace.KubeContext,
					}
				}
			}
		} else {
			if configDefinition != nil {
				log.Infof("Loaded config %s from %s", LoadedConfig, DefaultConfigsPath)
			} else {
				log.Infof("Loaded config %s", DefaultConfigPath)
			}
		}

		// Save generated config
		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatalf("Couldn't save generated config: %v", err)
		}
	})

	return config
}

// ValidateOnce ensures that specific values are set in the config
func ValidateOnce() {
	validateOnce.Do(func() {
		if config.Dev != nil {
			if config.Dev.Selectors != nil {
				for index, selectorConfig := range *config.Dev.Selectors {
					if selectorConfig.Name == nil {
						log.Fatalf("Error in config: Unnamed selector at index %d", index)
					}
				}
			}

			if config.Dev.Ports != nil {
				for index, port := range *config.Dev.Ports {
					if port.Selector == nil && port.LabelSelector == nil {
						log.Fatalf("Error in config: selector and label selector are nil in port config at index %d", index)
					}
					if port.PortMappings == nil {
						log.Fatalf("Error in config: portMappings is empty in port config at index %d", index)
					}
				}
			}

			if config.Dev.Sync != nil {
				for index, sync := range *config.Dev.Sync {
					if sync.Selector == nil && sync.LabelSelector == nil {
						log.Fatalf("Error in config: selector and label selector are nil in sync config at index %d", index)
					}
				}
			}

			if config.Dev.OverrideImages != nil {
				for index, overrideImageConfig := range *config.Dev.OverrideImages {
					if overrideImageConfig.Name == nil {
						log.Fatalf("Error in config: Unnamed override image config at index %d", index)
					}
				}
			}
		}

		if config.Deployments != nil {
			for index, deployConfig := range *config.Deployments {
				if deployConfig.Name == nil {
					log.Fatalf("Error in config: Unnamed deployment at index %d", index)
				}
				if deployConfig.Helm == nil && deployConfig.Kubectl == nil && deployConfig.Component == nil {
					log.Fatalf("Please specify either component, helm or kubectl as deployment type in deployment %s", *deployConfig.Name)
				}
				if deployConfig.Helm != nil && (deployConfig.Helm.Chart == nil || deployConfig.Helm.Chart.Name == nil) {
					log.Fatalf("deployments[%d].helm.chart and deployments[%d].helm.chart.name is required", index, index)
				}
				if deployConfig.Kubectl != nil && deployConfig.Kubectl.Manifests == nil {
					log.Fatalf("deployments[%d].kubectl.manifests is required", index)
				}
			}
		}
	})
}

func askQuestions(cache *generated.CacheConfig, vars []*configs.Variable) error {
	for idx, variable := range vars {
		if variable.Name == nil {
			return fmt.Errorf("Name required for variable with index %d", idx)
		}

		if _, ok := cache.Vars[*variable.Name]; ok {
			continue
		}

		cache.Vars[*variable.Name] = AskQuestion(variable)
	}

	return nil
}

// SetDevSpaceRoot checks the current directory and all parent directories for a .devspace folder with a config and sets the current working directory accordingly
func SetDevSpaceRoot() (bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return false, err
	}

	originalCwd := cwd
	homedir, err := homedir.Dir()
	if err != nil {
		return false, err
	}

	lastLength := 0
	for len(cwd) != lastLength {
		if cwd != homedir {
			configExists := configExistsInPath(cwd)
			if configExists {
				// Change working directory
				err = os.Chdir(cwd)
				if err != nil {
					return false, err
				}

				// Convert config if needed
				err = convertDotDevSpaceConfigToDevSpaceYaml(cwd)
				if err != nil {
					return false, errors.Wrap(err, "convert devspace config")
				}

				// Notify user that we are not using the current working directory
				if originalCwd != cwd {
					log.Infof("Using devspace config in %s", filepath.ToSlash(cwd))
				}

				return true, nil
			}
		}

		lastLength = len(cwd)
		cwd = filepath.Dir(cwd)
	}

	return false, nil
}

// GetSelector returns the service referenced by serviceName
func GetSelector(selectorName string) (*latest.SelectorConfig, error) {
	if config.Dev.Selectors != nil {
		for _, selector := range *config.Dev.Selectors {
			if *selector.Name == selectorName {
				return selector, nil
			}
		}
	}

	return nil, errors.New("Unable to find selector: " + selectorName)
}

// GetDefaultNamespace retrieves the default namespace where to operate in, either from devspace config or kube config
func GetDefaultNamespace(config *latest.Config) (string, error) {
	if config != nil && config.Cluster != nil && config.Cluster.Namespace != nil {
		return *config.Cluster.Namespace, nil
	}

	if config == nil || config.Cluster == nil || config.Cluster.APIServer == nil {
		kubeConfig, err := kubeconfig.ReadKubeConfig(clientcmd.RecommendedHomeFile)
		if err != nil {
			return "", err
		}

		activeContext := kubeConfig.CurrentContext
		if config != nil && config.Cluster != nil && config.Cluster.KubeContext != nil {
			activeContext = *config.Cluster.KubeContext
		}

		if kubeConfig.Contexts[activeContext] != nil && kubeConfig.Contexts[activeContext].Namespace != "" {
			return kubeConfig.Contexts[activeContext].Namespace, nil
		}
	}

	return "default", nil
}
