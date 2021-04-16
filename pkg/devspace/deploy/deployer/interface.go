package deployer

import (
	"io"
)

// Interface defines the common interface used for the deployment methods
type Interface interface {
	Status() (*StatusResult, error)
	Deploy(forceDeploy bool, builtImages map[string]string) (bool, error)
	Render(builtImages map[string]string, out io.Writer) error
	Delete() error
}

// StatusResult holds the status of a deployment
type StatusResult struct {
	Name   string
	Type   string
	Target string
	Status string
}
