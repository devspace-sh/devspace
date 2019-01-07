package generated

import (
	"io/ioutil"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"
)

// Config specifies the runtime config struct
type Config struct {
	HelmOverrideTimestamps map[string]int64  `yaml:"helmOverrideTimestamps"`
	HelmChartHashs         map[string]string `yaml:"helmChartHashs"`
	DockerfileTimestamps   map[string]int64  `yaml:"dockerfileTimestamps"`
	DockerContextPaths     map[string]string `yaml:"dockerContextPaths"`
	ImageTags              map[string]string `yaml:"imageTags"`
	Cloud                  *CloudConfig      `yaml:"cloud,omitempty"`
	ActiveConfig           *string           `yaml:"activeConfig,omitempty"`
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

// LoadConfig loads the config from the filesystem
func LoadConfig() (*Config, error) {
	workdir, _ := os.Getwd()

	data, err := ioutil.ReadFile(filepath.Join(workdir, ConfigPath))
	if err != nil {
		return &Config{
			DockerfileTimestamps:   make(map[string]int64),
			DockerContextPaths:     make(map[string]string),
			ImageTags:              make(map[string]string),
			HelmChartHashs:         make(map[string]string),
			HelmOverrideTimestamps: make(map[string]int64),
		}, nil
	}

	config := &Config{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	if config.HelmChartHashs == nil {
		config.HelmChartHashs = make(map[string]string)
	}
	if config.HelmOverrideTimestamps == nil {
		config.HelmOverrideTimestamps = make(map[string]int64)
	}
	if config.DockerfileTimestamps == nil {
		config.DockerfileTimestamps = make(map[string]int64)
	}
	if config.DockerContextPaths == nil {
		config.DockerContextPaths = make(map[string]string)
	}
	if config.ImageTags == nil {
		config.ImageTags = make(map[string]string)
	}

	return config, nil
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
