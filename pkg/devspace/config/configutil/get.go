package configutil

import (
	"os"
	"sync"

	"github.com/juju/errors"

	"github.com/covexo/devspace/pkg/util/kubeconfig"
	"github.com/covexo/devspace/pkg/util/log"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/covexo/devspace/pkg/devspace/config/v1"
)

//ConfigInterface defines the pattern of every config
type ConfigInterface interface{}

const configGitignore = `logs/
overwrite.yaml
generated.yaml
`

// DefaultConfigPath is the default config path to use
const DefaultConfigPath = ".devspace/config.yaml"

// DefaultOverwriteConfigPath is the default overwrite config path to use
const DefaultOverwriteConfigPath = ".devspace/overwrite.yaml"

// ConfigPath is the path for the main config
var ConfigPath = DefaultConfigPath

// OverwriteConfigPath specifies where the override.yaml lies
var OverwriteConfigPath = DefaultOverwriteConfigPath

// DefaultDevspaceServiceName is the name of the initial default service
const DefaultDevspaceServiceName = "default"

// DefaultDevspaceDeploymentName is the name of the initial default deployment
const DefaultDevspaceDeploymentName = "devspace-default"

// CurrentConfigVersion has the value of the current config version
const CurrentConfigVersion = "v1alpha1"

// Global config vars
var config *v1.Config          // merged config
var configRaw *v1.Config       // config from .devspace/config.yaml
var overwriteConfig *v1.Config // overwrite config from .devspace/config.yaml
var defaultConfig *v1.Config   // default config values

// Thread-safety helper
var getConfigOnce sync.Once
var setDefaultsOnce sync.Once

// ConfigExists checks whether the yaml file for the config exists
func ConfigExists() (bool, error) {
	workdir, _ := os.Getwd()

	_, err := os.Stat(workdir + ConfigPath)
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
	GetConfigWithoutDefaults()
	SetDefaultsOnce()

	return config
}

// GetConfigWithoutDefaults returns the config without setting the default values
func GetConfigWithoutDefaults() *v1.Config {
	getConfigOnce.Do(func() {
		config = makeConfig()
		overwriteConfig = makeConfig()
		configRaw = makeConfig()
		overwriteConfig = makeConfig()
		defaultConfig = makeConfig()

		err := loadConfig(configRaw, ConfigPath)
		if err != nil {
			log.Fatalf("Loading config: %v", err)
		}

		if configRaw.Version == nil || *configRaw.Version != CurrentConfigVersion {
			log.Fatal("Your config is out of date. Please run `devspace init -r` to update your config")
		}

		//ignore error as overwrite.yaml is optional
		loadConfig(overwriteConfig, OverwriteConfigPath)

		Merge(&config, configRaw, false)
		Merge(&config, overwriteConfig, true)
	})

	return config
}

// GetOverwriteConfig returns the config retrieved from .devspace/overwrite.yaml
func GetOverwriteConfig() *v1.Config {
	GetConfig()

	return overwriteConfig
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
