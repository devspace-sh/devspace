package types

import (
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
)

type Dependency interface {
	// Name will return the dependency name
	Name() string

	// Config holds the dependency config
	Config() config.Config

	// KubeClient returns the kube client of the dependency
	KubeClient() kubectl.Client

	// Children returns dependency children if any
	Children() []Dependency

	// Root determines if we are the top of the graph
	Root() bool

	// Path returns the folder where this dependency is stored
	Path() string

	// DependencyConfig is the config this dependency was created from
	DependencyConfig() *latest.DependencyConfig
}
