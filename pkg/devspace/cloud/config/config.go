package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

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

var loadedConfig *latest.Config
var loadedConfigErr error
var loadConfigOnce sync.Once

// GetProvider returns a provider from the loaded config
func GetProvider(config *latest.Config, provider string) *latest.Provider {
	for _, p := range config.Providers {
		if p.Name == provider {
			return p
		}
	}

	return nil
}

//Reset resets the loaded config and enables another loading processa
func Reset() {
	loadedConfig = nil
	loadedConfigErr = nil
	loadConfigOnce = sync.Once{}
}

// SaveProviderConfig saves the cloud config
func SaveProviderConfig(config *latest.Config) error {
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

// ParseProviderConfig reads the provider config and parses it
func ParseProviderConfig() (*latest.Config, error) {
	loadConfigOnce.Do(func() {
		loadedConfig, loadedConfigErr = loadProviderConfig()
	})

	return loadedConfig, loadedConfigErr
}

func loadProviderConfig() (*latest.Config, error) {
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

		return loadLegacyConfig(legacyPath)
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

func loadLegacyConfig(path string) (*latest.Config, error) {
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

	err = SaveProviderConfig(newConfig)
	if err != nil {
		return nil, errors.Wrap(err, "save config")
	}

	// Remove old config
	os.Remove(path)
	return newConfig, nil
}
