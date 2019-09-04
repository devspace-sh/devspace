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
}

// LastContextConfig holds all the informations about the last used kubernetes context
type LastContextConfig struct {
	Namespace string `yaml:"namespace,omitempty"`
	Context   string `yaml:"context,omitempty"`
}

// CacheConfig holds all the information specific to a certain config
type CacheConfig struct {
	Deployments  map[string]*DeploymentCache `yaml:"deployments,omitempty"`
	Images       map[string]*ImageCache      `yaml:"images,omitempty"`
	Dependencies map[string]string           `yaml:"dependencies,omitempty"`
	Vars         map[string]string           `yaml:"vars,omitempty"`
	LastContext  *LastContextConfig          `yaml:"lastContext,omitempty"`
}

// ImageCache holds the cache related information about a certain image
type ImageCache struct {
	ImageConfigHash string `yaml:"imageConfigHash,omitempty"`

	DockerfileHash string `yaml:"dockerfileHash,omitempty"`
	ContextHash    string `yaml:"contextHash,omitempty"`
	EntrypointHash string `yaml:"entrypointHash,omitempty"`

	CustomFilesHash string `yaml:"customFilesHash,omitempty"`

	ImageName string `yaml:"imageName,omitempty"`
	Tag       string `yaml:"tag,omitempty"`
}

// DeploymentCache holds the information about a specific deployment
type DeploymentCache struct {
	DeploymentConfigHash string `yaml:"deploymentConfigHash,omitempty"`

	HelmOverridesHash    string `yaml:"helmOverridesHash,omitempty"`
	HelmChartHash        string `yaml:"helmChartHash,omitempty"`
	KubectlManifestsHash string `yaml:"kubectlManifestsHash,omitempty"`
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

//ResetConfig resets the config to nil and enables loading from configs.yaml
func ResetConfig() {
	loadedConfigOnce = sync.Once{}
	loadedConfig = nil
}

// LoadConfig loads the config from the filesystem
func LoadConfig() (*Config, error) {
	var err error

	loadedConfigOnce.Do(func() {
		loadedConfig, err = LoadConfigFromPath(ConfigPath)
	})

	return loadedConfig, err
}

// LoadConfigFromPath loads the generated config from a given path
func LoadConfigFromPath(path string) (*Config, error) {
	var loadedConfig *Config

	data, readErr := ioutil.ReadFile(path)
	if readErr != nil {
		loadedConfig = &Config{
			ActiveConfig: DefaultConfigName,
			Configs:      make(map[string]*CacheConfig),
		}
	} else {
		loadedConfig = &Config{}
		err := yaml.Unmarshal(data, loadedConfig)
		if err != nil {
			return nil, err
		}

		if loadedConfig.ActiveConfig == "" {
			loadedConfig.ActiveConfig = DefaultConfigName
		}
		if loadedConfig.Configs == nil {
			loadedConfig.Configs = make(map[string]*CacheConfig)
		}
	}

	InitDevSpaceConfig(loadedConfig, loadedConfig.ActiveConfig)
	return loadedConfig, nil
}

// NewCache returns a new cache object
func NewCache() *CacheConfig {
	return &CacheConfig{
		Deployments: make(map[string]*DeploymentCache),
		Images:      make(map[string]*ImageCache),

		Dependencies: make(map[string]string),
		Vars:         make(map[string]string),
	}
}

// GetActive returns the currently active devspace config
func (config *Config) GetActive() *CacheConfig {
	InitDevSpaceConfig(config, config.ActiveConfig)
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
		config.Configs[configName] = NewCache()
		return
	}

	if config.Configs[configName].Deployments == nil {
		config.Configs[configName].Deployments = make(map[string]*DeploymentCache)
	}
	if config.Configs[configName].Images == nil {
		config.Configs[configName].Images = make(map[string]*ImageCache)
	}
	if config.Configs[configName].Dependencies == nil {
		config.Configs[configName].Dependencies = make(map[string]string)
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

	InitDevSpaceConfig(config, config.ActiveConfig)

	configPath := filepath.Join(workdir, ConfigPath)
	err = os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(configPath, data, 0666)
}
