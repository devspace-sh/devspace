package dependency

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
)

// Dependency holds the dependency config and has an id
type Dependency struct {
	ID              string
	LocalPath       string
	Config          *latest.Config
	GeneratedConfig *generated.Config

	DependencyConfig *latest.DependencyConfig
	DependencyCache  *generated.CacheConfig
}

// Deploy deploys the dependency if necessary
func (d *Dependency) Deploy() {

}

// Purge purges the dependency
func (d *Dependency) Purge() {

}
