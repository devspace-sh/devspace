package generated

import (
	"io/ioutil"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"
)

// Config specifies the runtime config struct
type Config struct {
	HelmChartHash          string            `yaml:"chartHash"`
	DockerLatestTimestamps map[string]int64  `yaml:"dockerLatestTimestamps"`
	ImageTags              map[string]string `yaml:"imageTags"`
}

// ConfigPath is the relative generated config path
var ConfigPath = "/.devspace/generated.yaml"

// LoadConfig loads the config from the filesystem
func LoadConfig() (*Config, error) {
	workdir, _ := os.Getwd()

	data, err := ioutil.ReadFile(filepath.Join(workdir, ConfigPath))
	if err != nil {
		return &Config{
			DockerLatestTimestamps: make(map[string]int64),
			ImageTags:              make(map[string]string),
		}, nil
	}

	config := &Config{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
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
