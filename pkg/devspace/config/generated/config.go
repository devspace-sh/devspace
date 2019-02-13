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
	ActiveConfig string                     `yaml:"activeConfig,omitempty"`
	Configs      map[string]*DevSpaceConfig `yaml:"configs,omitempty"`
	// Key is ProviderName:SpaceID
	Spaces map[string]*SpaceConfig `yaml:"space,omitempty"`
}

// DevSpaceConfig holds all the information specific to a certain config
type DevSpaceConfig struct {
	Deployments          map[string]*DeploymentConfig `yaml:"deployments"`
	DockerfileTimestamps map[string]int64             `yaml:"dockerfileTimestamps"`
	DockerContextPaths   map[string]string            `yaml:"dockerContextPaths"`
	ImageTags            map[string]string            `yaml:"imageTags"`
	Vars                 map[string]interface{}       `yaml:"vars,omitempty"`
	// ProviderName:SpaceID
	SpaceID *string `yaml:"spaceID,omitempty"`
}

// DeploymentConfig holds the information about a specific deployment
type DeploymentConfig struct {
	HelmOverrideTimestamps map[string]int64 `yaml:"helmOverrideTimestamps"`
	HelmChartHash          string           `yaml:"helmChartHash"`
}

// SpaceConfig holds the information about a space in the cloud
type SpaceConfig struct {
	SpaceID             int     `yaml:"spaceID"`
	ProviderName        string  `yaml:"providerName"`
	Name                string  `yaml:"name"`
	Namespace           string  `yaml:"namespace"`
	Created             string  `yaml:"created"`
	ServiceAccountToken string  `yaml:"serviceAccountToken"`
	CaCert              string  `yaml:"caCert"`
	Server              string  `yaml:"server"`
	Domain              *string `yaml:"domain"`
}

// ConfigPath is the relative generated config path
var ConfigPath = "/.devspace/generated.yaml"

var loadedConfig *Config
var loadedConfigOnce sync.Once

// LoadConfig loads the config from the filesystem
func LoadConfig() (*Config, error) {
	var err error

	loadedConfigOnce.Do(func() {
		workdir, _ := os.Getwd()

		data, err := ioutil.ReadFile(filepath.Join(workdir, ConfigPath))
		if err != nil {
			loadedConfig = &Config{
				ActiveConfig: DefaultConfigName,
				Configs:      make(map[string]*DevSpaceConfig),
				Spaces:       make(map[string]*SpaceConfig),
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
				loadedConfig.Configs = make(map[string]*DevSpaceConfig)
			}
			if loadedConfig.Spaces == nil {
				loadedConfig.Spaces = make(map[string]*SpaceConfig)
			}
		}

		InitDevSpaceConfig(loadedConfig, loadedConfig.ActiveConfig)
	})

	return loadedConfig, err
}

// GetActive returns the currently active devspace config
func (config *Config) GetActive() *DevSpaceConfig {
	return config.Configs[config.ActiveConfig]
}

// InitDevSpaceConfig verifies a given config name is set
func InitDevSpaceConfig(config *Config, configName string) {
	if _, ok := config.Configs[configName]; ok == false {
		config.Configs[configName] = &DevSpaceConfig{
			DockerfileTimestamps: make(map[string]int64),
			DockerContextPaths:   make(map[string]string),
			ImageTags:            make(map[string]string),
			Deployments:          make(map[string]*DeploymentConfig),
			Vars:                 make(map[string]interface{}),
		}

		return
	}

	if config.Configs[configName].DockerfileTimestamps == nil {
		config.Configs[configName].DockerfileTimestamps = make(map[string]int64)
	}
	if config.Configs[configName].DockerContextPaths == nil {
		config.Configs[configName].DockerContextPaths = make(map[string]string)
	}
	if config.Configs[configName].ImageTags == nil {
		config.Configs[configName].ImageTags = make(map[string]string)
	}
	if config.Configs[configName].Deployments == nil {
		config.Configs[configName].Deployments = make(map[string]*DeploymentConfig)
	}
	if config.Configs[configName].Vars == nil {
		config.Configs[configName].Vars = make(map[string]interface{})
	}
}

// SaveConfig saves the config to the filesystem
func SaveConfig(config *Config) error {
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
