package configutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	yaml "gopkg.in/yaml.v2"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/juju/errors"

	"github.com/covexo/devspace/pkg/util/kubeconfig"
	"github.com/covexo/devspace/pkg/util/log"

	"github.com/covexo/devspace/pkg/devspace/config/generated"
	v1 "github.com/covexo/devspace/pkg/devspace/config/v1"
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

// DefaultOverwriteConfigPath is the default overwrite config path to use
const DefaultOverwriteConfigPath = ".devspace/overwrite.yaml"

// ConfigPath is the path for the main config or if a configs.yaml is there the config to load
var ConfigPath = DefaultConfigPath

// OverwriteConfigPath specifies where the override.yaml lies
var OverwriteConfigPath = DefaultOverwriteConfigPath

// LoadedConfig is the config that was loaded from the configs file
var LoadedConfig string

// DefaultDevspaceServiceName is the name of the initial default service
const DefaultDevspaceServiceName = "default"

// DefaultDevspaceDeploymentName is the name of the initial default deployment
const DefaultDevspaceDeploymentName = "devspace-default"

// CurrentConfigVersion has the value of the current config version
const CurrentConfigVersion = "v1alpha1"

// Global config vars
var config *v1.Config          // merged config
var configRaw *v1.Config       // config from .devspace/config.yaml
var overwriteConfig *v1.Config // overwrite config from .devspace/overwrite.yaml
var defaultConfig *v1.Config   // default config values

// Thread-safety helper
var getConfigOnce sync.Once
var setDefaultsOnce sync.Once

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
func InitConfig() *v1.Config {
	getConfigOnce.Do(func() {
		config = makeConfig()
		overwriteConfig = makeConfig()
		configRaw = makeConfig()
		overwriteConfig = makeConfig()
		defaultConfig = makeConfig()
	})

	return config
}

// GetBaseConfig returns the config unmerged with potential overwrites
func GetBaseConfig() *v1.Config {
	GetConfigWithoutDefaults(false)
	SetDefaultsOnce()

	return config
}

// GetConfig returns the config merged with all potential overwrite files
func GetConfig() *v1.Config {
	GetConfigWithoutDefaults(true)
	SetDefaultsOnce()

	return config
}

// GetConfigWithoutDefaults returns the config without setting the default values
func GetConfigWithoutDefaults(loadOverwrites bool) *v1.Config {
	getConfigOnce.Do(func() {
		var configDefinition *v1.ConfigDefinition

		// Init configs
		config = makeConfig()
		configRaw = makeConfig()
		overwriteConfig = makeConfig()
		defaultConfig = makeConfig()

		// Get generated config
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatalf("Error loading %s: %v", generated.ConfigPath, err)
		}

		// Check if configs.yaml exists
		_, err = os.Stat(DefaultConfigsPath)
		if err == nil {
			configs := v1.Configs{}

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
				log.Fatalf("config key not defined in config %s", LoadedConfig)
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
				vars := []*v1.Variable{}
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

			err = loadConfigFromPath(configRaw, ConfigPath)
			if err != nil {
				log.Fatalf("Loading config: %v", err)
			}
		}

		// Check if version key is defined
		if configRaw.Version == nil {
			log.Fatalf("The version key is missing in your config. Current config version is %s", CurrentConfigVersion)
		}

		// Check config version
		if *configRaw.Version != CurrentConfigVersion {
			log.Fatal("Your config is out of date. Please run `devspace init -r` to update your config")
		}

		Merge(&config, deepCopy(configRaw))

		// Check if we should load overwrites
		if loadOverwrites {
			if configDefinition == nil {
				//ignore error as overwrite.yaml is optional
				err := loadConfigFromPath(overwriteConfig, OverwriteConfigPath)
				if err != nil {
					Merge(&config, overwriteConfig)
					log.Infof("Loaded config %s with overwrite config %s", ConfigPath, OverwriteConfigPath)
				} else {
					log.Infof("Loaded config %s", ConfigPath)
				}
			} else if configDefinition.Overwrites != nil {
				for index, configWrapper := range *configDefinition.Overwrites {
					overwriteConfig, err := loadConfigFromWrapper(configWrapper)
					if err != nil {
						log.Fatalf("Error loading overwrite config at index %d: %v", index, err)
					}

					Merge(&config, overwriteConfig)
				}

				log.Infof("Loaded config %s from %s with %d overwrites", LoadedConfig, DefaultConfigsPath, len(*configDefinition.Overwrites))
			} else {
				log.Infof("Loaded config %s from %s without overwrites", LoadedConfig, DefaultConfigsPath)
			}
		} else {
			if configDefinition == nil {
				log.Infof("Loaded config %s", ConfigPath)
			} else {
				log.Infof("Loaded config %s from %s", LoadedConfig, DefaultConfigsPath)
			}
		}
	})

	return config
}

// SetDefaultsOnce ensures that specific values are set in the config
func SetDefaultsOnce() {
	setDefaultsOnce.Do(func() {
		// Initialize Namespaces
		if config.DevSpace != nil {
			if config.DevSpace.Deployments != nil {
				for index, deployConfig := range *config.DevSpace.Deployments {
					if deployConfig.Name == nil {
						log.Fatalf("Error in config: Unnamed deployment at index %d", index)
					}

					if deployConfig.Namespace == nil {
						deployConfig.Namespace = String("")
					}
				}
			}

			if config.DevSpace.Services != nil {
				for index, serviceConfig := range *config.DevSpace.Services {
					if serviceConfig.Name == nil {
						log.Fatalf("Error in config: Unnamed service at index %d", index)
					}

					if serviceConfig.Namespace == nil {
						serviceConfig.Namespace = String("")
					}
				}
			}

			if config.DevSpace.Sync != nil {
				for _, syncPath := range *config.DevSpace.Sync {
					if syncPath.Namespace == nil {
						syncPath.Namespace = String("")
					}
				}
			}

			if config.DevSpace.Ports != nil {
				for _, portForwarding := range *config.DevSpace.Ports {
					if portForwarding.Namespace == nil {
						portForwarding.Namespace = String("")
					}
				}
			}
		}

		if config.Images != nil {
			for _, buildConfig := range *config.Images {
				if buildConfig.Build != nil && buildConfig.Build.Kaniko != nil {
					if buildConfig.Build.Kaniko.Namespace == nil {
						buildConfig.Build.Kaniko.Namespace = String("")
					}
				}
			}
		}
	})
}

func askQuestions(generatedConfig *generated.Config, vars []*v1.Variable) error {
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

// GetService returns the service referenced by serviceName
func GetService(serviceName string) (*v1.ServiceConfig, error) {
	if config.DevSpace.Services != nil {
		for _, service := range *config.DevSpace.Services {
			if *service.Name == serviceName {
				return service, nil
			}
		}
	}
	return nil, errors.New("Unable to find service: " + serviceName)
}

// AddService adds a service to the config
func AddService(service *v1.ServiceConfig) error {
	if config.DevSpace.Services == nil {
		config.DevSpace.Services = &[]*v1.ServiceConfig{}
	}
	*config.DevSpace.Services = append(*config.DevSpace.Services, service)

	return nil
}

// GetDefaultNamespace retrieves the default namespace where to operate in, either from devspace config or kube config
func GetDefaultNamespace(config *v1.Config) (string, error) {
	if config.Cluster != nil && config.Cluster.Namespace != nil {
		return *config.Cluster.Namespace, nil
	}

	if config.Cluster == nil || config.Cluster.APIServer == nil {
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
