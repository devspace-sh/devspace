package deployer

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"io"
)

// Interface defines the common interface used for the deployment methods
type Interface interface {
	Status(ctx devspacecontext.Context) (*StatusResult, error)
	Deploy(ctx devspacecontext.Context, forceDeploy bool) (bool, error)
	Render(ctx devspacecontext.Context, out io.Writer) error
}

// StatusResult holds the status of a deployment
type StatusResult struct {
	Name   string
	Type   string
	Target string
	Status string
}
