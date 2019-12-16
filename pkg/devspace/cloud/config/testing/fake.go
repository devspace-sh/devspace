package testing

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	
	"github.com/pkg/errors"
)

type cloudLoader struct {
	config *latest.Config
}

// NewLoader creates a new instance of the interface Loader
func NewLoader(config *latest.Config) config.Loader {
	return &cloudLoader{config}
} 

// Save saves the cloud config
func (l *cloudLoader) Save(config *latest.Config) error {
	return nil
}

// Load reads the provider config and parses it
func (l *cloudLoader) Load() (*latest.Config, error) {
	if l.config == nil {
		return nil, errors.New("Couldn't load cloud config")
	}

	return l.config, nil
}

// GetDefaultProviderName returns the default provider name
func (l *cloudLoader) GetDefaultProviderName() (string, error) {
	return l.config.Default, nil
}
