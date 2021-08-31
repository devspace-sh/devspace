package types

import (
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
)

type Dependency interface {
	// ID returns the id of the dependency
	ID() string

	// Name will return the dependency name
	Name() string

	// Config holds the dependency config
	Config() config.Config

	// Children returns dependency children if any
	Children() []Dependency

	// Root returns if the dependency is a direct dependency of the root DevSpace config
	Root() bool

	// LocalPath returns the path where this dependency is stored
	LocalPath() string

	// BuiltImages returns the images that were built by this dependency
	BuiltImages() map[string]string

	// DependencyConfig is the config this dependency was created from
	DependencyConfig() *latest.DependencyConfig

	// ReplacePods replaces the dependencies pods from dev.replacePods
	ReplacePods(client kubectl.Client, logger log.Logger) error

	// StartSync starts the dependency sync
	StartSync(client kubectl.Client, interrupt chan error, printSyncLog, verboseSync bool, logger log.Logger) error

	// StartPortForwarding starts the dependency port-forwarding
	StartPortForwarding(client kubectl.Client, interrupt chan error, logger log.Logger) error

	// StartReversePortForwarding starts the dependency port-forwarding
	StartReversePortForwarding(client kubectl.Client, interrupt chan error, logger log.Logger) error
}
