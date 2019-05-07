package generated

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	yaml "gopkg.in/yaml.v2"
)

// DefaultConfigName is the default
const DefaultConfigName = "default"

// Config specifies the runtime config struct
type Config struct {
	ActiveConfig string                  `yaml:"activeConfig,omitempty"`
	Configs      map[string]*CacheConfig `yaml:"configs,omitempty"`
	CloudSpace   *CloudSpaceConfig       `yaml:"space"`
}

// CloudSpaceConfig holds all the informations about a certain cloud space
type CloudSpaceConfig struct {
	SpaceID      int    `yaml:"spaceID"`
	OwnerID      int    `yaml:"ownerID"`
	Owner        string `yaml:"owner"`
	ProviderName string `yaml:"providerName"`
	KubeContext  string `yaml:"kubeContext"`
	Name         string `yaml:"name"`
	Created      string `yaml:"created"`
}

// CacheConfig holds all the information specific to a certain config
type CacheConfig struct {
	Deployments map[string]*DeploymentCache `yaml:"deployments"`
	Images      map[string]*ImageCache      `yaml:"images"`
	Vars        map[string]string           `yaml:"vars,omitempty"`
}

// ImageCache holds the cache related information about a certain image
type ImageCache struct {
	ImageConfigHash string `yaml:"imageConfigHash"`

	DockerfileHash string `yaml:"dockerfileHash"`
	ContextHash    string `yaml:"contextHash"`
	EntrypointHash string `yaml:"entrypointHash"`

	ImageName string `yaml:"imageName"`
	Tag       string `yaml:"tag"`
}

// DeploymentCache holds the information about a specific deployment
type DeploymentCache struct {
	DeploymentConfigHash string `yaml:"deploymentConfigHash"`

	HelmOverridesHash    string `yaml:"helmOverridesHash"`
	HelmChartHash        string `yaml:"helmChartHash"`
	KubectlManifestsHash string `yaml:"kubectlManifestsHash"`
}

// ConfigPath is the relative generated config path
var ConfigPath = ".devspace/generated.yaml"

var loadedConfig *Config
var loadedConfigOnce sync.Once

var testDontSaveConfig = false

// SetTestConfig sets the config for testing purposes
func SetTestConfig(config *Config) {
	loadedConfigOnce.Do(func() {})
	loadedConfig = config
	testDontSaveConfig = true
}

// LoadConfig loads the config from the filesystem
func LoadConfig() (*Config, error) {
	var err error

	loadedConfigOnce.Do(func() {
		data, readErr := ioutil.ReadFile(ConfigPath)
		if readErr != nil {
			loadedConfig = &Config{
				ActiveConfig: DefaultConfigName,
				Configs:      make(map[string]*CacheConfig),
			}
		} else {
			loadedConfig = &Config{}
			err = yaml.Unmarshal(data, loadedConfig)
			if err != nil {
				return
			}

			if loadedConfig.ActiveConfig == "" {
				loadedConfig.ActiveConfig = DefaultConfigName
			}
			if loadedConfig.Configs == nil {
				loadedConfig.Configs = make(map[string]*CacheConfig)
			}
		}

		InitDevSpaceConfig(loadedConfig, loadedConfig.ActiveConfig)
	})

	return loadedConfig, err
}

// GetActive returns the currently active devspace config
func (config *Config) GetActive() *CacheConfig {
	return config.Configs[config.ActiveConfig]
}

// GetImageCache returns the image cache if it exists and creates one if not
func (cache *CacheConfig) GetImageCache(imageConfigName string) *ImageCache {
	if _, ok := cache.Images[imageConfigName]; !ok {
		cache.Images[imageConfigName] = &ImageCache{}
	}

	return cache.Images[imageConfigName]
}

// GetDeploymentCache returns the deployment cache if it exists and creates one if not
func (cache *CacheConfig) GetDeploymentCache(deploymentName string) *DeploymentCache {
	if _, ok := cache.Deployments[deploymentName]; !ok {
		cache.Deployments[deploymentName] = &DeploymentCache{}
	}

	return cache.Deployments[deploymentName]
}

// InitDevSpaceConfig verifies a given config name is set
func InitDevSpaceConfig(config *Config, configName string) {
	if _, ok := config.Configs[configName]; ok == false {
		config.Configs[configName] = &CacheConfig{
			Deployments: make(map[string]*DeploymentCache),
			Images:      make(map[string]*ImageCache),
			Vars:        make(map[string]string),
		}

		return
	}

	if config.Configs[configName].Deployments == nil {
		config.Configs[configName].Deployments = make(map[string]*DeploymentCache)
	}
	if config.Configs[configName].Images == nil {
		config.Configs[configName].Images = make(map[string]*ImageCache)
	}
	if config.Configs[configName].Vars == nil {
		config.Configs[configName].Vars = make(map[string]string)
	}
}

// SaveConfig saves the config to the filesystem
func SaveConfig(config *Config) error {
	if testDontSaveConfig {
		return nil
	}

	workdir, _ := os.Getwd()

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	configPath := filepath.Join(workdir, ConfigPath)

	err = os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(configPath, data, 0666)
}
