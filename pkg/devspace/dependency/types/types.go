package types

import (
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

type Dependency interface {
	// ID returns the id of the dependency
	ID() string

	// Name will return the dependency name
	Name() string

	// Children returns dependency children if any
	Children() []Dependency

	// Root returns if the dependency is a direct dependency of the root DevSpace config
	Root() bool

	// LocalPath returns the path where this dependency is stored
	LocalPath() string

	// BuiltImages returns the images that were built by this dependency
	BuiltImages() map[string]string

	// Config holds the dependency config
	Config() config.Config

	// DependencyConfig is the config this dependency was created from
	DependencyConfig() *latest.DependencyConfig
}
