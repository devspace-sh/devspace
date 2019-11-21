package generated

import (
	"io/ioutil"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"
)

// ConfigPath is the relative generated config path
var ConfigPath = ".devspace/generated.yaml"

// ConfigLoader is the interface for loading the generated config
type ConfigLoader interface {
	Load() (*Config, error)
	LoadFromPath(path string) (*Config, error)
	Save(config *Config) error
}

type configLoader struct {
	profile string
}

// NewConfigLoader creates a new generated config loader
func NewConfigLoader(profile string) ConfigLoader {
	return &configLoader{
		profile: profile,
	}
}

// Load loads the config from the filesystem
func (l *configLoader) Load() (*Config, error) {
	return l.LoadFromPath(ConfigPath)
}

// LoadFromPath loads the generated config from a given path
func (l *configLoader) LoadFromPath(path string) (*Config, error) {
	var loadedConfig *Config

	data, readErr := ioutil.ReadFile(path)
	if readErr != nil {
		loadedConfig = &Config{
			OverrideProfile: nil,
			ActiveProfile:   "",
			Profiles:        make(map[string]*CacheConfig),
			Vars:            make(map[string]string),
		}
	} else {
		loadedConfig = &Config{}
		err := yaml.Unmarshal(data, loadedConfig)
		if err != nil {
			return nil, err
		}

		if loadedConfig.Profiles == nil {
			loadedConfig.Profiles = make(map[string]*CacheConfig)
		}
		if loadedConfig.Vars == nil {
			loadedConfig.Vars = make(map[string]string)
		}
	}

	// Set override profile
	if l.profile != "" {
		loadedConfig.OverrideProfile = &l.profile
	} else {
		loadedConfig.OverrideProfile = nil
	}

	return loadedConfig, nil
}

// Save saves the config to the filesystem
func (l *configLoader) Save(config *Config) error {
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

// NewCache returns a new cache object
func NewCache() *CacheConfig {
	return &CacheConfig{
		Deployments: make(map[string]*DeploymentCache),
		Images:      make(map[string]*ImageCache),

		Dependencies: make(map[string]string),
	}
}

// GetActiveProfile returns the active profile
func (config *Config) GetActiveProfile() string {
	active := config.ActiveProfile
	if config.OverrideProfile != nil {
		active = *config.OverrideProfile
	}

	return active
}

// GetActive returns the currently active devspace config
func (config *Config) GetActive() *CacheConfig {
	active := config.GetActiveProfile()

	InitDevSpaceConfig(config, active)
	return config.Profiles[active]
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
	if _, ok := config.Profiles[configName]; ok == false {
		config.Profiles[configName] = NewCache()
		return
	}

	if config.Profiles[configName].Deployments == nil {
		config.Profiles[configName].Deployments = make(map[string]*DeploymentCache)
	}
	if config.Profiles[configName].Images == nil {
		config.Profiles[configName].Images = make(map[string]*ImageCache)
	}
	if config.Profiles[configName].Dependencies == nil {
		config.Profiles[configName].Dependencies = make(map[string]string)
	}
}
