package configutil

import (
	"os"
	"sync"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/juju/errors"

	"github.com/covexo/devspace/pkg/util/kubeconfig"
	"github.com/covexo/devspace/pkg/util/log"

	"github.com/covexo/devspace/pkg/devspace/config/generated"
	v1 "github.com/covexo/devspace/pkg/devspace/config/v1"
)

//ConfigInterface defines the pattern of every config
type ConfigInterface interface{}

const configGitignore = `logs/
overwrite.yaml
generated.yaml
`

// DefaultCloudTarget is the default cloud target to use
const DefaultCloudTarget = "dev"

// DefaultConfigsPath is the default configs path to use
const DefaultConfigsPath = ".devspace/configs.yaml"

// DefaultConfigPath is the default config path to use
const DefaultConfigPath = ".devspace/config.yaml"

// DefaultOverwriteConfigPath is the default overwrite config path to use
const DefaultOverwriteConfigPath = ".devspace/overwrite.yaml"

// ConfigPath is the path for the main config
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
func ConfigExists() (bool, error) {
	_, err := os.Stat(DefaultConfigsPath)
	if err == nil {
		return true, nil // configs.yaml found
	}

	_, err = os.Stat(ConfigPath)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
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

// GetConfig returns the config merged from .devspace/config.yaml and .devspace/overwrite.yaml
func GetConfig() *v1.Config {
	GetConfigWithoutDefaults(true)
	SetDefaultsOnce()

	return config
}

// GetConfigWithoutDefaults returns the config without setting the default values
func GetConfigWithoutDefaults(loadOverwrites bool) *v1.Config {
	getConfigOnce.Do(func() {
		var configDefinition *v1.ConfigDefinition

		config = makeConfig()
		configRaw = makeConfig()
		overwriteConfig = makeConfig()
		defaultConfig = makeConfig()

		// Check if configs.yaml exists
		_, err := os.Stat(DefaultConfigsPath)
		if err == nil {
			configs := v1.Configs{}

			// Get generated config
			generatedConfig, err := generated.LoadConfig()
			if err != nil {
				log.Fatalf("Error loading %s: %v", generated.ConfigPath, err)
			}

			err = loadConfigs(&configs, DefaultConfigsPath)
			if err != nil {
				log.Fatalf("Error loading %s: %v", DefaultConfigsPath, err)
			}

			if generatedConfig.ActiveConfig == nil || *generatedConfig.ActiveConfig == "" {
				// check if default config exists
				if configs["default"] == nil {
					if len(configs) == 0 {
						log.Fatalf("No config found in %s", DefaultConfigsPath)
					}

					for name := range configs {
						LoadedConfig = name
						break
					}
				} else {
					LoadedConfig = "default"
				}
			}

			configDefinition = configs[LoadedConfig]
			if configDefinition.Config == nil {
				log.Fatalf("config key not defined in config %s", LoadedConfig)
			}

			configRaw, err = loadConfigFromWrapper(configDefinition.Config)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			err = loadConfig(configRaw, ConfigPath)
			if err != nil {
				log.Fatalf("Loading config: %v", err)
			}
		}

		// Check config version
		if configRaw.Version == nil || *configRaw.Version != CurrentConfigVersion {
			log.Fatal("Your config is out of date. Please run `devspace init -r` to update your config")
		}

		Merge(&config, deepCopy(configRaw))

		// Check if we should load overwrites
		if loadOverwrites {
			if configDefinition == nil {
				//ignore error as overwrite.yaml is optional
				loadConfig(overwriteConfig, OverwriteConfigPath)

				Merge(&config, overwriteConfig)
				return
			}

			if configDefinition.Overwrites != nil && len(*configDefinition.Overwrites) > 0 {
				for index, configWrapper := range *configDefinition.Overwrites {
					overwriteConfig, err := loadConfigFromWrapper(configWrapper)
					if err != nil {
						log.Fatalf("Error loading overwrite config at index %d: %v", index, err)
					}

					Merge(&config, overwriteConfig)
				}
			}
		}
	})

	return config
}

// SetDefaultsOnce ensures that specific values are set in the config
func SetDefaultsOnce() {
	setDefaultsOnce.Do(func() {
		defaultNamespace, err := GetDefaultNamespace(config)
		if err != nil {
			log.Fatalf("Error retrieving default namespace: %v", err)
		}

		// Initialize Namespaces
		if config.DevSpace != nil {
			needTiller := config.InternalRegistry != nil

			if config.DevSpace.Deployments != nil {
				for index, deployConfig := range *config.DevSpace.Deployments {
					if deployConfig.Name == nil {
						log.Fatalf("Error in config: Unnamed deployment at index %d", index)
					}

					if deployConfig.Namespace == nil {
						deployConfig.Namespace = String("")
					}

					if deployConfig.Helm != nil {
						needTiller = true
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

			if needTiller && config.Tiller == nil {
				defaultConfig.Tiller = &v1.TillerConfig{
					Namespace: &defaultNamespace,
				}

				config.Tiller = defaultConfig.Tiller
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

		if config.InternalRegistry != nil {
			defaultConfig.InternalRegistry = &v1.InternalRegistryConfig{
				Namespace: &defaultNamespace,
			}

			config.InternalRegistry.Namespace = defaultConfig.InternalRegistry.Namespace
		}
	})
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

// GetCurrentCloudTarget retrieves the current cloud target from the config
func GetCurrentCloudTarget(config *v1.Config) *string {
	if config.Cluster == nil || config.Cluster.CloudProvider == nil || *config.Cluster.CloudProvider == "" {
		return nil
	}

	if config.Cluster.CloudTarget == nil {
		return String(DefaultCloudTarget)
	}

	return config.Cluster.CloudTarget
}
