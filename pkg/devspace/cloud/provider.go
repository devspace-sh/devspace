package cloud

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/client"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
)

// Provider interacts with one cloud provider
type Provider interface {
	GetAndUpdateSpaceCache(spaceID int, forceUpdate bool) (*latest.SpaceCache, bool, error)
	CacheSpace(space *latest.Space, serviceAccount *latest.ServiceAccount) error

	ConnectCluster(options *ConnectClusterOptions) error
	ResetKey(clusterName string) error

	UpdateKubeConfig(contextName string, serviceAccount *latest.ServiceAccount, spaceID int, setActive bool) error
	DeleteKubeContext(space *latest.Space) error

	GetClusterKey(cluster *latest.Cluster) (string, error)
	AskForEncryptionKey(cluster *latest.Cluster) (string, error)

	PrintToken(spaceID int) error
	PrintSpaces(cluster, name string, all bool) error

	Save() error
	Client() client.Client
	GetConfig() latest.Provider
}

// DevSpaceKubeContextName is the name for the kube config context
const DevSpaceKubeContextName = "devspace"

// Provider describes the struct to hold the cloud configuration
type provider struct {
	latest.Provider

	client client.Client
	log    log.Logger
}

// GetProvider returns the current specified cloud provider
func GetProvider(useProviderName *string, log log.Logger) (Provider, error) {
	// Get provider configuration
	providerConfig, err := config.ParseProviderConfig()
	if err != nil {
		return nil, err
	}

	providerName := config.DevSpaceCloudProviderName
	if useProviderName == nil {
		// Choose cloud provider
		if providerConfig.Default != "" {
			providerName = providerConfig.Default
		} else if len(providerConfig.Providers) > 1 {
			options := []string{}
			for _, providerHost := range providerConfig.Providers {
				options = append(options, providerHost.Name)
			}

			providerName, err = survey.Question(&survey.QuestionOptions{
				Question: "Select cloud provider",
				Options:  options,
			}, log)
			if err != nil {
				return nil, err
			}
		}
	} else {
		providerName = *useProviderName
	}

	// Ensure user is logged in
	err = EnsureLoggedIn(providerConfig, providerName, log)
	if err != nil {
		return nil, err
	}

	// Set cluster key map
	p := config.GetProvider(providerConfig, providerName)
	if p.ClusterKey == nil {
		p.ClusterKey = make(map[int]string)
	}

	client := client.NewClient(providerName, p.Host, p.Key, p.Token)

	// Return provider config
	return &provider{*p, client, log}, nil
}

// Save saves the provider config
func (p *provider) Save() error {
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

// Client returns the providers' client
func (p *provider) Client() client.Client {
	return p.client
}

// Client returns the providers' client
func (p *provider) GetConfig() latest.Provider {
	return p.Provider
}
