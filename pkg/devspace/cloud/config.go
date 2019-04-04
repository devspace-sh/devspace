package cloud

import (
	"os"

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
	Name  string `yaml:"name,omitempty"`
	Host  string `yaml:"host,omitempty"`
	Token string `yaml:"token,omitempty"`
}

// DevSpaceCloudProviderConfig holds the information for the devspace-cloud
var DevSpaceCloudProviderConfig = &Provider{
	Name: DevSpaceCloudProviderName,
	Host: "https://app.devspace.cloud",
}

// ParseCloudConfig parses the cloud configuration and returns a map containing the configurations
func ParseCloudConfig() (ProviderConfig, error) {
	data, err := config.ReadCloudsConfig()
	if os.IsNotExist(err) {
		return ProviderConfig{
			DevSpaceCloudProviderName: DevSpaceCloudProviderConfig,
		}, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "read clouds config")
	}

	cloudConfig := make(ProviderConfig)
	err = yaml.Unmarshal(data, cloudConfig)
	if err != nil {
		return nil, err
	}

	if _, ok := cloudConfig[DevSpaceCloudProviderName]; ok {
		cloudConfig[DevSpaceCloudProviderName].Host = DevSpaceCloudProviderConfig.Host
	} else {
		cloudConfig[DevSpaceCloudProviderName] = DevSpaceCloudProviderConfig
	}

	for configName, config := range cloudConfig {
		config.Name = configName
	}

	return cloudConfig, nil
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
			Name:  "",
			Host:  host,
			Token: provider.Token,
		}
	}

	out, err := yaml.Marshal(saveConfig)
	if err != nil {
		return err
	}

	return config.SaveCloudsConfig(out)
}
