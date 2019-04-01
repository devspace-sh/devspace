package deploy

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
)

// Interface defines the common interface used for the deployment methods
type Interface interface {
	Delete() error
	Status() (*StatusResult, error)
	Deploy(generatedConfig *generated.Config, isDev, forceDeploy bool) error
}

// StatusResult holds the status of a deployment
type StatusResult struct {
	Name   string
	Type   string
	Target string
	Status string
}
