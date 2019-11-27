package testing

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// FakeController is the fake build controller
type FakeController struct{}

// NewFakeController creates a new fake build controller
func NewFakeController(config *latest.Config) deploy.Controller {
	return &FakeController{}
}

// Deploy deploys the deployments
func (f *FakeController) Deploy(options *deploy.Options, log log.Logger) error {
	return nil
}

// Purge purges the deployments
func (f *FakeController) Purge(deployments []string, log log.Logger) error {
	return nil
}
