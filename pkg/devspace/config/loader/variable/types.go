package variable

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
)

// Variable defines an interface to load a variable
type Variable interface {
	Load(ctx context.Context, definition *latest.Variable) (interface{}, error)
}

// RuntimeResolver fills in runtime variables and cached ones
type RuntimeResolver interface {
	// FillRuntimeVariables finds the used variables first and then fills in those in the haystack
	FillRuntimeVariables(haystack interface{}, config config.Config, dependencies []types.Dependency) (interface{}, error)

	// FillRuntimeVariablesWithRebuild finds the used variables first and then fills in those in the haystack
	FillRuntimeVariablesWithRebuild(haystack interface{}, config config.Config, dependencies []types.Dependency, builtImages map[string]string) (bool, interface{}, error)
}

// Resolver defines an interface to resolve defined variables
type Resolver interface {
	// DefinedVars returns the defined variables
	DefinedVars() map[string]*latest.Variable

	// UpdateVars sets the defined variables to use in the resolver
	UpdateVars(vars map[string]*latest.Variable)

	// FindVariables returns all variable names that were found in the given map
	FindVariables(haystack interface{}) ([]*latest.Variable, error)

	// FillVariables finds the used variables first and then fills in those in the haystack
	FillVariables(ctx context.Context, haystack interface{}, skipUnused bool) (interface{}, error)

	// FillVariablesExclude finds the used variables first and then fills in those that do not match the excluded paths in the haystack
	FillVariablesExclude(ctx context.Context, haystack interface{}, skipUnused bool, excluded []string) (interface{}, error)

	// FillVariablesInclude finds the used variables first and then fills in those that match the included paths in the haystack
	FillVariablesInclude(ctx context.Context, haystack interface{}, skipUnused bool, included []string) (interface{}, error)

	// ResolvedVariables returns the internal memory cache of the resolver with all resolved variables
	ResolvedVariables() map[string]interface{}
}
