package dependency

import (
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
)

// Dependency holds the dependency config and has an id
type Dependency struct {
	name         string
	absolutePath string
	root         bool
	localConfig  config.Config

	children []types.Dependency

	dependencyConfig *latest.DependencyConfig
	dependencyCache  localcache.Cache

	kubeClient kubectl.Client
}

// Implement Interface Methods

func (d *Dependency) Name() string { return d.name }

func (d *Dependency) Root() bool { return d.root }

func (d *Dependency) KubeClient() kubectl.Client { return d.kubeClient }

func (d *Dependency) Config() config.Config { return d.localConfig }

func (d *Dependency) Path() string { return d.absolutePath }

func (d *Dependency) DependencyConfig() *latest.DependencyConfig { return d.dependencyConfig }

func (d *Dependency) Children() []types.Dependency { return d.children }

func skipDependency(name string, skipDependencies []string) bool {
	for _, sd := range skipDependencies {
		if sd == name {
			return true
		}
	}
	return false
}

func foundDependency(name string, dependencies []string) bool {
	if len(dependencies) == 0 {
		return true
	}

	for _, n := range dependencies {
		if n == name {
			return true
		}
	}

	return false
}
