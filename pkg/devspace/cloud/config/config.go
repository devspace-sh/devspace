package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/legacy"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// DevSpaceCloudProviderName is the name of the default devspace-cloud provider
const DevSpaceCloudProviderName = "app.devspace.cloud"

// LegacyDevSpaceCloudConfigPath holds the path to the cloud config file
var LegacyDevSpaceCloudConfigPath = constants.DefaultHomeDevSpaceFolder + "/clouds.yaml"

// DevSpaceProvidersConfigPath is the path to the providers config
var DevSpaceProvidersConfigPath = constants.DefaultHomeDevSpaceFolder + "/providers.yaml"

// DevSpaceCloudProviderConfig holds the information for the devspace-cloud
var DevSpaceCloudProviderConfig = &latest.Provider{
	Name: DevSpaceCloudProviderName,
	Host: "https://app.devspace.cloud",
}

// Loader saves and loads cloud configuration
type Loader interface {
	Save(config *latest.Config) error
	Load() (*latest.Config, error)
	GetDefaultProviderName() (string, error)
}

type loader struct {
	loadedConfig    *latest.Config
	loadedConfigErr error
}

// NewLoader creates a new instance of the interface Loader
func NewLoader() Loader {
	return &loader{}
}

// GetProvider returns a provider from the loaded config
func GetProvider(config *latest.Config, provider string) *latest.Provider {
	for _, p := range config.Providers {
		if p.Name == provider {
			return p
		}
	}

	return nil
}

// Save saves the cloud config
func (l *loader) Save(config *latest.Config) error {
	homedir, err := homedir.Dir()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	cfgPath := filepath.Join(homedir, DevSpaceProvidersConfigPath)
	err = os.MkdirAll(filepath.Dir(cfgPath), 0755)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(cfgPath, data, 0600)
}

// Load reads the provider config and parses it
func (l *loader) Load() (*latest.Config, error) {
	if l.loadedConfig == nil && l.loadedConfigErr == nil {
		l.loadedConfig, l.loadedConfigErr = l.loadProviderConfig()
	}

	return l.loadedConfig, l.loadedConfigErr
}

func (l *loader) loadProviderConfig() (*latest.Config, error) {
	homedir, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homedir, DevSpaceProvidersConfigPath)
	_, err = os.Stat(configPath)
	if os.IsNotExist(err) {
		// Check for legacy config
		legacyPath := filepath.Join(homedir, LegacyDevSpaceCloudConfigPath)
		_, err = os.Stat(legacyPath)
		if os.IsNotExist(err) {
			return &latest.Config{
				Version: latest.Version,
				Providers: []*latest.Provider{
					DevSpaceCloudProviderConfig,
				},
			}, nil
		} else if err != nil {
			return nil, err
		}

		return l.loadLegacyConfig(legacyPath)
	} else if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	config := &latest.Config{
		Version:   latest.Version,
		Providers: []*latest.Provider{},
	}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	// Ensure default config is there
	defaultConfigFound := false
	for _, provider := range config.Providers {
		if provider.Name == DevSpaceCloudProviderName {
			defaultConfigFound = true
		}
		if provider.Host == "" {
			provider.Host = DevSpaceCloudProviderConfig.Host
		}
	}
	if !defaultConfigFound {
		config.Providers = append(config.Providers, DevSpaceCloudProviderConfig)
	}

	return config, nil
}

func (l *loader) loadLegacyConfig(path string) (*latest.Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := make(legacy.Config)
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	if _, ok := config[DevSpaceCloudProviderName]; ok {
		if config[DevSpaceCloudProviderName].Host == "" {
			config[DevSpaceCloudProviderName].Host = DevSpaceCloudProviderConfig.Host
		}
	} else {
		config[DevSpaceCloudProviderName] = &legacy.Provider{
			Name: DevSpaceCloudProviderName,
			Host: "https://app.devspace.cloud",
		}
	}

	newConfig := &latest.Config{
		Version:   latest.Version,
		Providers: []*latest.Provider{},
	}

	for configName, config := range config {
		config.Name = configName
		if config.ClusterKey == nil {
			config.ClusterKey = make(map[int]string)
		}

		newConfig.Providers = append(newConfig.Providers, &latest.Provider{
			Name:       config.Name,
			Host:       config.Host,
			Key:        config.Key,
			Token:      config.Token,
			ClusterKey: config.ClusterKey,
		})
	}

	err = l.Save(newConfig)
	if err != nil {
		return nil, errors.Wrap(err, "save config")
	}

	// Remove old config
	os.Remove(path)
	return newConfig, nil
}

// GetDefaultProviderName returns the default provider name
func (l *loader) GetDefaultProviderName() (string, error) {
	// Get provider configuration
	providerConfig, err := l.Load()
	if err != nil {
		return "", err
	}

	// Choose cloud provider
	providerName := DevSpaceCloudProviderName
	if providerConfig.Default != "" {
		providerName = providerConfig.Default
	}

	return providerName, nil
}
