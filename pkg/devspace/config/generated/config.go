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
	Space        *SpaceConfig               `yaml:"space,omitempty"`
}

// DevSpaceConfig holds all the information specific to a certain config
type DevSpaceConfig struct {
	Dev    CacheConfig            `yaml:"dev,omitempty"`
	Deploy CacheConfig            `yaml:"deploy,omitempty"`
	Vars   map[string]interface{} `yaml:"vars,omitempty"`
}

// CacheConfig holds the information if things have to be redeployed or rebuild
type CacheConfig struct {
	Deployments          map[string]*DeploymentConfig `yaml:"deployments"`
	DockerfileTimestamps map[string]int64             `yaml:"dockerfileTimestamps"`
	DockerContextPaths   map[string]string            `yaml:"dockerContextPaths"`
	ImageTags            map[string]string            `yaml:"imageTags"`
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
			Dev: CacheConfig{
				Deployments:          make(map[string]*DeploymentConfig),
				DockerfileTimestamps: make(map[string]int64),
				DockerContextPaths:   make(map[string]string),
				ImageTags:            make(map[string]string),
			},
			Deploy: CacheConfig{
				Deployments:          make(map[string]*DeploymentConfig),
				DockerfileTimestamps: make(map[string]int64),
				DockerContextPaths:   make(map[string]string),
				ImageTags:            make(map[string]string),
			},
			Vars: make(map[string]interface{}),
		}

		return
	}

	if config.Configs[configName].Dev.DockerfileTimestamps == nil {
		config.Configs[configName].Dev.DockerfileTimestamps = make(map[string]int64)
	}
	if config.Configs[configName].Deploy.DockerfileTimestamps == nil {
		config.Configs[configName].Deploy.DockerfileTimestamps = make(map[string]int64)
	}
	if config.Configs[configName].Dev.DockerContextPaths == nil {
		config.Configs[configName].Dev.DockerContextPaths = make(map[string]string)
	}
	if config.Configs[configName].Deploy.DockerContextPaths == nil {
		config.Configs[configName].Deploy.DockerContextPaths = make(map[string]string)
	}
	if config.Configs[configName].Dev.ImageTags == nil {
		config.Configs[configName].Dev.ImageTags = make(map[string]string)
	}
	if config.Configs[configName].Deploy.ImageTags == nil {
		config.Configs[configName].Deploy.ImageTags = make(map[string]string)
	}
	if config.Configs[configName].Dev.Deployments == nil {
		config.Configs[configName].Dev.Deployments = make(map[string]*DeploymentConfig)
	}
	if config.Configs[configName].Deploy.Deployments == nil {
		config.Configs[configName].Deploy.Deployments = make(map[string]*DeploymentConfig)
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
