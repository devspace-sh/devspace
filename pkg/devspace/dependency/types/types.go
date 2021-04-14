package types

import (
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

type Dependency interface {
	// ID returns the id of the dependency
	ID() string

	// NameOrID will return either the dependency name (if it has any) or the ID if not
	NameOrID() string

	// LocalPath returns the path where this dependency is stored
	LocalPath() string

	// Config holds the dependency config
	Config() config.Config

	// DependencyConfig is the config this dependency was created from
	DependencyConfig() *latest.DependencyConfig
}
