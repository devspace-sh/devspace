package cloud

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/client"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/pkg/errors"
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
	loader config.Loader
	log    log.Logger
}

// GetProvider returns the current specified cloud provider
func GetProvider(useProviderName string, log log.Logger) (Provider, error) {
	// Get provider configuration
	loader := config.NewLoader()

	return GetProviderWithOptions(useProviderName, "", false, loader, log)
}

// GetProviderWithOptions returns a provider by options
func GetProviderWithOptions(useProviderName, key string, relogin bool, loader config.Loader, log log.Logger) (Provider, error) {
	var err error

	//Get config
	providerConfig, err := loader.Load()
	if err != nil {
		return nil, err
	}

	// Get provider name
	providerName := config.DevSpaceCloudProviderName
	if useProviderName == "" {
		// Choose cloud provider
		if providerConfig.Default != "" {
			providerName = providerConfig.Default
		} else if len(providerConfig.Providers) > 1 {
			options := []string{}
			for _, providerHost := range providerConfig.Providers {
				options = append(options, providerHost.Name)
			}

			providerName, err = log.Question(&survey.QuestionOptions{
				Question: "Select cloud provider",
				Options:  options,
			})
			if err != nil {
				return nil, err
			}
		}
	} else {
		providerName = useProviderName
	}

	// Let's check if we are logged in first
	p := config.GetProvider(providerConfig, providerName)
	if p == nil {
		cloudProviders := ""
		for _, p := range providerConfig.Providers {
			cloudProviders += p.Name + " "
		}

		return nil, errors.Errorf("Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: %s", cloudProviders)
	}

	provider := &provider{
		*p,
		client.NewClient(providerName, p.Host, p.Key, p.Token),
		loader,
		log,
	}
	if relogin == true || provider.Key == "" {
		provider.Token = ""
		provider.Key = ""

		if key != "" {
			provider.Key = key

			// Check if we got access
			_, err := provider.client.GetSpaces()
			if err != nil {
				return nil, errors.Errorf("Access denied for key %s: %v", key, err)
			}
		} else {
			err := provider.Login()
			if err != nil {
				return nil, errors.Wrap(err, "Login")
			}
		}

		log.Donef("Successfully logged into %s", provider.Name)

		// Login into registries
		err = provider.loginIntoRegistries()
		if err != nil {
			log.Warnf("Error logging into docker registries: %v", err)
		}

		err = provider.Save()
		if err != nil {
			return nil, err
		}
	}

	// Return provider config
	return provider, nil
}

// Save saves the provider config
func (p *provider) Save() error {
	providerConfig, err := p.loader.Load()
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

	return p.loader.Save(providerConfig)
}

// Client returns the providers' client
func (p *provider) Client() client.Client {
	return p.client
}

// Client returns the providers' client
func (p *provider) GetConfig() latest.Provider {
	return p.Provider
}
