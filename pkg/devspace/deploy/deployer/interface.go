package deployer

import (
	"io"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
)

// Interface defines the common interface used for the deployment methods
type Interface interface {
	Status() (*StatusResult, error)
	Deploy(cache *generated.CacheConfig, forceDeploy bool, builtImages map[string]string) (bool, error)
	Render(cache *generated.CacheConfig, builtImages map[string]string, out io.Writer) error
	Delete(cache *generated.CacheConfig) error
}

// StatusResult holds the status of a deployment
type StatusResult struct {
	Name   string
	Type   string
	Target string
	Status string
}
