package client

import (
	"io/ioutil"
	"net/http"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/pkg/errors"
)

// TokenEndpoint is the endpoint where to get a token from
const TokenEndpoint = "/auth/token"

// GetToken returns a valid access token to the provider
func (c *client) GetToken() (string, error) {
	if c.accessKey == "" {
		return "", errors.New("Provider has no key specified")
	}
	if c.token != "" && token.IsTokenValid(c.token) {
		return c.token, nil
	}

	resp, err := http.Get(c.host + TokenEndpoint + "?key=" + c.accessKey)
	if err != nil {
		return "", errors.Wrap(err, "token request")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "read request body")
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("Error retrieving token: Code %v => %s. Try to relogin with 'devspace login'", resp.StatusCode, string(body))
	}

	c.token = string(body)
	if token.IsTokenValid(c.token) == false {
		return "", errors.New("Received invalid token from provider")
	}

	err = c.saveToken()
	if err != nil {
		return "", errors.Wrap(err, "token save")
	}

	return c.token, nil
}

func (c *client) saveToken() error {
	loader := config.NewLoader()
	providerConfig, err := loader.Load()
	if err != nil {
		return err
	}

	for idx, provider := range providerConfig.Providers {
		if provider.Name == c.provider {
			providerConfig.Providers[idx].Token = c.token
			return loader.Save(providerConfig)
		}
	}

	return errors.Errorf("Couldn't find provider %s", c.provider)
}
