package cloud

import (
	"os"
	"sync"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/pkg/errors"

	yaml "gopkg.in/yaml.v2"
)

// ProviderConfig holds all the different providers and their configuration
type ProviderConfig map[string]*Provider

// DevSpaceCloudProviderName is the name of the default devspace-cloud provider
const DevSpaceCloudProviderName = "app.devspace.cloud"

// DevSpaceKubeContextName is the name for the kube config context
const DevSpaceKubeContextName = "devspace"

// GraphqlEndpoint is the endpoint where to execute graphql requests
const GraphqlEndpoint = "/graphql"

// Provider describes the struct to hold the cloud configuration
type Provider struct {
	Name string `yaml:"name,omitempty"`
	Host string `yaml:"host,omitempty"`

	// Key is used to obtain a token from the auth server
	Key string `yaml:"key,omitempty"`

	// Token is the actual authorization bearer
	Token string `yaml:"token,omitempty"`

	ClusterKey map[int]string `yaml:"clusterKeys,omitempty"`
}

// DevSpaceCloudProviderConfig holds the information for the devspace-cloud
var DevSpaceCloudProviderConfig = &Provider{
	Name: DevSpaceCloudProviderName,
	Host: "https://app.devspace.cloud",
}

var loadedConfig ProviderConfig
var loadedConfigOnce sync.Once

// LoadCloudConfig parses the cloud configuration and returns a map containing the configurations
func LoadCloudConfig() (ProviderConfig, error) {
	var err error

	loadedConfigOnce.Do(func() {
		var data []byte

		data, err = config.ReadCloudsConfig()
		if os.IsNotExist(err) {
			loadedConfig = ProviderConfig{
				DevSpaceCloudProviderName: DevSpaceCloudProviderConfig,
			}

			err = nil
			return
		} else if err != nil {
			err = errors.Wrap(err, "read clouds config")
			return
		}

		loadedConfig = make(ProviderConfig)
		err = yaml.Unmarshal(data, loadedConfig)
		if err != nil {
			return
		}

		if _, ok := loadedConfig[DevSpaceCloudProviderName]; ok {
			loadedConfig[DevSpaceCloudProviderName].Host = DevSpaceCloudProviderConfig.Host
		} else {
			loadedConfig[DevSpaceCloudProviderName] = DevSpaceCloudProviderConfig
		}

		for configName, config := range loadedConfig {
			config.Name = configName
			if config.ClusterKey == nil {
				config.ClusterKey = make(map[int]string)
			}
		}
	})

	return loadedConfig, err
}

// SaveCloudConfig saves the provider configuration to file
func SaveCloudConfig(providerConfig ProviderConfig) error {
	saveConfig := ProviderConfig{}

	for name, provider := range providerConfig {
		host := provider.Host
		if name == DevSpaceCloudProviderName {
			host = ""
		}

		saveConfig[name] = &Provider{
			Name:       "",
			Host:       host,
			Key:        provider.Key,
			Token:      provider.Token,
			ClusterKey: provider.ClusterKey,
		}
	}

	out, err := yaml.Marshal(saveConfig)
	if err != nil {
		return err
	}

	return config.SaveCloudsConfig(out)
}

// Save saves the provider config
func (p *Provider) Save() error {
	providerConfig, err := LoadCloudConfig()
	if err != nil {
		return errors.Wrap(err, "load cloud config")
	}

	// Make sure provider is set
	providerConfig[p.Name] = p
	return SaveCloudConfig(providerConfig)
}
