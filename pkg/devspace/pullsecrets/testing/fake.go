package testing

import "github.com/loft-sh/devspace/pkg/devspace/pullsecrets"

// Client is a fake implementation of the Client interface
type Client struct{}

// CreatePullSecrets is a fake implementation of the function
func (c *Client) CreatePullSecrets() error {
	return nil
}

// CreatePullSecret is a fake implementation of the function
func (c *Client) CreatePullSecret(options *pullsecrets.PullSecretOptions) error {
	return nil
}
