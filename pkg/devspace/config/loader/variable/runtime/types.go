package runtime

import (
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/imageselector"
)

// RuntimeResolver fills in runtime variables and cached ones
type RuntimeResolver interface {
	// FillRuntimeVariablesAsImageSelector finds the used variables first and then fills in those in the haystack
	FillRuntimeVariablesAsImageSelector(haystack interface{}, config config.Config, dependencies []types.Dependency) (*imageselector.ImageSelector, error)

	// FillRuntimeVariablesAsString finds the used variables first and then fills in those in the haystack
	FillRuntimeVariablesAsString(haystack interface{}, config config.Config, dependencies []types.Dependency) (string, error)

	// FillRuntimeVariables finds the used variables first and then fills in those in the haystack
	FillRuntimeVariables(haystack interface{}, config config.Config, dependencies []types.Dependency) (interface{}, error)

	// FillRuntimeVariablesWithRebuild finds the used variables first and then fills in those in the haystack
	FillRuntimeVariablesWithRebuild(haystack interface{}, config config.Config, dependencies []types.Dependency, builtImages map[string]string) (bool, interface{}, error)
}
