package cloud

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/pkg/errors"
)

// loginIntoRegistries logs the user into the user docker registries
func (p *provider) loginIntoRegistries() error {
	registries, err := p.client.GetRegistries()
	if err != nil {
		return errors.Wrap(err, "get registries")
	}

	// We don't want the minikube client to login into the registry
	client, err := docker.NewClient(p.log)
	if err != nil {
		return errors.Wrap(err, "new docker client")
	}

	// Get token
	bearerToken, err := p.client.GetToken()
	if err != nil {
		return errors.Wrap(err, "get token")
	}

	// Get account name
	accountName, err := token.GetAccountName(bearerToken)
	if err != nil {
		return errors.Wrap(err, "get account name")
	}

	for _, registry := range registries {
		// Login
		_, err = client.Login(registry.URL, accountName, p.Key, true, true, true)
		if err != nil {
			return errors.Wrap(err, "docker login")
		}

		p.log.Donef("Successfully logged into docker registry %s", registry.URL)
		p.log.Infof("You can now use %s/%s/* to deploy private docker images", registry.URL, accountName)
	}

	return nil
}
