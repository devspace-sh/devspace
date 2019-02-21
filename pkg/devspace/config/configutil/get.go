package configutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	yaml "gopkg.in/yaml.v2"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/juju/errors"
	homedir "github.com/mitchellh/go-homedir"

	"github.com/covexo/devspace/pkg/util/kubeconfig"
	"github.com/covexo/devspace/pkg/util/log"

	"github.com/covexo/devspace/pkg/devspace/config/configs"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/devspace/config/versions/latest"
)

//ConfigInterface defines the pattern of every config
type ConfigInterface interface{}

// DefaultCloudTarget is the default cloud target to use
const DefaultCloudTarget = "dev"

// DefaultConfigsPath is the default configs path to use
const DefaultConfigsPath = ".devspace/configs.yaml"

// DefaultVarsPath is the default vars path to use if no configs.yaml is present
const DefaultVarsPath = ".devspace/vars.yaml"

// DefaultConfigPath is the default config path to use
const DefaultConfigPath = ".devspace/config.yaml"

// ConfigPath is the path for the main config or if a configs.yaml is there the config to load
var ConfigPath = DefaultConfigPath

// LoadedConfig is the config that was loaded from the configs file
var LoadedConfig string

// DefaultDevspaceServiceName is the name of the initial default service
const DefaultDevspaceServiceName = "default"

// DefaultDevspaceDeploymentName is the name of the initial default deployment
const DefaultDevspaceDeploymentName = "devspace-app"

// Global config vars
var config *latest.Config    // merged config
var configRaw *latest.Config // config from .devspace/config.yaml

// Thread-safety helper
var getConfigOnce sync.Once
var validateOnce sync.Once

// ConfigExists checks whether the yaml file for the config exists or the configs.yaml exists
func ConfigExists() bool {
	// Check configs.yaml
	_, err := os.Stat(DefaultConfigsPath)
	if err == nil {
		return true // configs.yaml found
	}

	// Check normal config.yaml
	_, err = os.Stat(ConfigPath)
	if err != nil {
		return false
	}

	return true // Normal config file found
}

// InitConfig initializes the config objects
func InitConfig() *latest.Config {
	getConfigOnce.Do(func() {
		config = latest.New().(*latest.Config)
		configRaw = latest.New().(*latest.Config)
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

// GetConfigWithoutDefaults returns the config without setting the default values
func GetConfigWithoutDefaults(loadOverwrites bool) *latest.Config {
	getConfigOnce.Do(func() {
		var configDefinition *configs.ConfigDefinition

		// Init configs
		config = latest.New().(*latest.Config)
		configRaw = latest.New().(*latest.Config)

		// Get generated config
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatalf("Error loading %s: %v", generated.ConfigPath, err)
		}

		// Check if configs.yaml exists
		_, err = os.Stat(DefaultConfigsPath)
		if err == nil {
			configs := configs.Configs{}

			// Get configs
			err = LoadConfigs(&configs, DefaultConfigsPath)
			if err != nil {
				log.Fatalf("Error loading %s: %v", DefaultConfigsPath, err)
			}

			// Get config to load
			if generatedConfig.ActiveConfig == "" {
				// check if default config exists
				if configs[generated.DefaultConfigName] == nil {
					if len(configs) == 0 {
						log.Fatalf("No config found in %s", DefaultConfigsPath)
					}

					for name := range configs {
						LoadedConfig = name
						break
					}
				} else {
					LoadedConfig = generated.DefaultConfigName
				}
			} else {
				LoadedConfig = generatedConfig.ActiveConfig
			}

			// Check if we should override loadedconfig
			if ConfigPath != DefaultConfigPath {
				LoadedConfig = ConfigPath
			}

			// Get real config definition
			configDefinition = configs[LoadedConfig]
			if configDefinition.Config == nil {
				log.Fatalf("config %s cannot be found", LoadedConfig)
			}

			// Ask questions
			if configDefinition.Vars != nil {
				vars, err := loadVarsFromWrapper(configDefinition.Vars)
				if err != nil {
					log.Fatalf("Error loading vars: %v", err)
				}

				err = askQuestions(generatedConfig, vars)
				if err != nil {
					log.Fatalf("Error filling vars: %v", err)
				}
			}

			// Load config
			configRaw, err = loadConfigFromWrapper(configDefinition.Config)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			_, err := os.Stat(DefaultVarsPath)
			if err == nil {
				vars := []*configs.Variable{}
				yamlFileContent, err := ioutil.ReadFile(DefaultVarsPath)
				if err != nil {
					log.Fatalf("Error loading %s: %v", DefaultVarsPath, err)
				}

				err = yaml.UnmarshalStrict(yamlFileContent, vars)
				if err != nil {
					log.Fatalf("Error parsing %s: %v", DefaultVarsPath, err)
				}

				// Ask questions
				err = askQuestions(generatedConfig, vars)
				if err != nil {
					log.Fatalf("Error filling vars: %v", err)
				}
			}

			configRaw, err = loadConfigFromPath(ConfigPath)
			if err != nil {
				log.Fatalf("Loading config: %v", err)
			}
		}

		Merge(&config, deepCopy(configRaw))

		// Check if we should load overwrites
		if loadOverwrites {
			if configDefinition != nil && configDefinition.Overwrites != nil {
				for index, configWrapper := range *configDefinition.Overwrites {
					overwriteConfig, err := loadConfigFromWrapper(configWrapper)
					if err != nil {
						log.Fatalf("Error loading overwrite config at index %d: %v", index, err)
					}

					Merge(&config, overwriteConfig)
				}

				log.Infof("Loaded config %s from %s with %d overwrites", LoadedConfig, DefaultConfigsPath, len(*configDefinition.Overwrites))
			} else {
				log.Infof("Loaded config from %s", DefaultConfigsPath)
			}
		} else {
			if configDefinition != nil {
				log.Infof("Loaded config %s from %s", LoadedConfig, DefaultConfigsPath)
			} else {
				log.Infof("Loaded config %s", ConfigPath)
			}
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
					if sync.ContainerPath == nil || sync.LocalSubPath == nil {
						log.Fatalf("Error in config: containerPath or localSubPath are nil in sync config at index %d", index)
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
				if deployConfig.Helm == nil && deployConfig.Kubectl == nil {
					log.Fatalf("Please specify either helm or kubectl as deployment type in deployment %s", *deployConfig.Name)
				}
				if deployConfig.Helm != nil && deployConfig.Helm.ChartPath == nil {
					log.Fatalf("deployments[%d].helm.chartPath is required", index)
				}
				if deployConfig.Kubectl != nil && deployConfig.Kubectl.Manifests == nil {
					log.Fatalf("deployments[%d].kubectl.manifests is required", index)
				}
			}
		}
	})
}

func askQuestions(generatedConfig *generated.Config, vars []*configs.Variable) error {
	changed := false
	activeConfig := generatedConfig.GetActive()

	for idx, variable := range vars {
		if variable.Name == nil {
			return fmt.Errorf("Name required for variable with index %d", idx)
		}

		if _, ok := activeConfig.Vars[*variable.Name]; ok {
			continue
		}

		activeConfig.Vars[*variable.Name] = AskQuestion(variable)
		changed = true
	}

	if changed {
		err := generated.SaveConfig(generatedConfig)
		if err != nil {
			return err
		}
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
			_, err := os.Stat(filepath.Join(cwd, ".devspace"))
			if err == nil {
				// Change working directory
				err = os.Chdir(cwd)
				if err != nil {
					return false, err
				}

				// Notify user that we are not using the current working directory
				if originalCwd != cwd {
					log.Infof("Using devspace config in %s/.devspace", filepath.ToSlash(cwd))
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
		if config.Cluster.KubeContext != nil {
			activeContext = *config.Cluster.KubeContext
		}

		if kubeConfig.Contexts[activeContext] != nil && kubeConfig.Contexts[activeContext].Namespace != "" {
			return kubeConfig.Contexts[activeContext].Namespace, nil
		}
	}

	return "default", nil
}
