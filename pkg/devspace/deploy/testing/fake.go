package testing

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
)

// FakeController is the fake build controller
type FakeController struct{}

// NewFakeController creates a new fake build controller
func NewFakeController(config *latest.Config) deploy.Controller {
	return &FakeController{}
}

// Deploy deploys the deployments
func (f *FakeController) Deploy(ctx devspacecontext.Context, deployments []string, options *deploy.Options) error {
	return nil
}

// Purge purges the deployments
func (f *FakeController) Purge(ctx devspacecontext.Context, deployments []string, options *deploy.PurgeOptions) error {
	return nil
}
