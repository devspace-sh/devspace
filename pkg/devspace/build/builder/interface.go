package builder

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
)

// Interface defines methods for builders docker, kaniko and custom
type Interface interface {
	ShouldRebuild(ctx devspacecontext.Context, forceRebuild bool) (bool, error)
	Build(ctx devspacecontext.Context) error
}
