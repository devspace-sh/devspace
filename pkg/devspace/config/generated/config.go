package generated

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	yaml "gopkg.in/yaml.v2"
)

// Config specifies the runtime config struct
type Config struct {
	HelmOverrideTimestamps map[string]int64       `yaml:"helmOverrideTimestamps"`
	HelmChartHashs         map[string]string      `yaml:"helmChartHashs"`
	DockerfileTimestamps   map[string]int64       `yaml:"dockerfileTimestamps"`
	DockerContextPaths     map[string]string      `yaml:"dockerContextPaths"`
	ImageTags              map[string]string      `yaml:"imageTags"`
	Cloud                  *CloudConfig           `yaml:"cloud,omitempty"`
	ActiveConfig           *string                `yaml:"activeConfig,omitempty"`
	Vars                   map[string]interface{} `yaml:"vars,omitempty"`
}

// DevSpaceTargetConfig holds the information to connect to a devspace target
type DevSpaceTargetConfig struct {
	TargetName          string
	Namespace           string
	ServiceAccountToken string
	CaCert              string
	Server              string

	Domain *string
}

// CloudConfig holds the information to authenticate with the cloud provider
type CloudConfig struct {
	DevSpaceID   int                              `yaml:"devSpaceID"`
	ProviderName string                           `yaml:"providerName"`
	Name         string                           `yaml:"name"`
	Targets      map[string]*DevSpaceTargetConfig `yaml:"targets"`
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
				DockerfileTimestamps:   make(map[string]int64),
				DockerContextPaths:     make(map[string]string),
				ImageTags:              make(map[string]string),
				HelmChartHashs:         make(map[string]string),
				HelmOverrideTimestamps: make(map[string]int64),
				Vars:                   make(map[string]interface{}),
			}

			return
		}

		loadedConfig = &Config{}
		err = yaml.Unmarshal(data, loadedConfig)
		if err != nil {
			return
		}

		if loadedConfig.HelmChartHashs == nil {
			loadedConfig.HelmChartHashs = make(map[string]string)
		}
		if loadedConfig.HelmOverrideTimestamps == nil {
			loadedConfig.HelmOverrideTimestamps = make(map[string]int64)
		}
		if loadedConfig.DockerfileTimestamps == nil {
			loadedConfig.DockerfileTimestamps = make(map[string]int64)
		}
		if loadedConfig.DockerContextPaths == nil {
			loadedConfig.DockerContextPaths = make(map[string]string)
		}
		if loadedConfig.ImageTags == nil {
			loadedConfig.ImageTags = make(map[string]string)
		}
		if loadedConfig.ImageTags == nil {
			loadedConfig.Vars = make(map[string]interface{})
		}
	})

	return loadedConfig, err
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
