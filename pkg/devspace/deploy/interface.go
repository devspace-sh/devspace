package deploy

import (
	"github.com/covexo/devspace/pkg/devspace/config/generated"
)

// Interface defines the common interface used for the deployment methods
type Interface interface {
	Delete() error
	Status() ([][]string, error)
	Deploy(generatedConfig *generated.Config, forceDeploy, useDevOverwrite bool) error
}
