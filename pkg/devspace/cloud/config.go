package cloud

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// DevSpaceKubeContextName is the name for the kube config context
const DevSpaceKubeContextName = "devspace"

// GraphqlEndpoint is the endpoint where to execute graphql requests
const GraphqlEndpoint = "/graphql"

// Provider describes the struct to hold the cloud configuration
type Provider struct {
	latest.Provider

	Log log.Logger
}

// Save saves the provider config
func (p *Provider) Save() error {
	providerConfig, err := config.ParseProviderConfig()
	if err != nil {
		return err
	}

	found := false
	for idx, provider := range providerConfig.Providers {
		if provider.Name == p.Name {
			found = true
			providerConfig.Providers[idx] = &p.Provider
			break
		}
	}
	if !found {
		providerConfig.Providers = append(providerConfig.Providers, &p.Provider)
	}

	return config.SaveProviderConfig(providerConfig)
}
