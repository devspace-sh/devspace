package testing

import "github.com/devspace-cloud/devspace/pkg/devspace/registry"

// Client is a fake implementation of the Client interface
type Client struct{}

// CreatePullSecrets is a fake implementation of the function
func (c *Client) CreatePullSecrets() error {
	return nil
}

// CreatePullSecret is a fake implementation of the function
func (c *Client) CreatePullSecret(options *registry.PullSecretOptions) error {
	return nil
}
