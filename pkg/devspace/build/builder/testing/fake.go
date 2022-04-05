package testing

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
)

// Builder is a fake implementation of the Interface interface
type Builder struct {
}

// ShouldRebuild is a fake implementation of the function
func (b *Builder) ShouldRebuild(ctx devspacecontext.Context, forceRebuild bool) (bool, error) {
	return forceRebuild, nil
}

// Build is a fake implementation of the function
func (b *Builder) Build(ctx devspacecontext.Context) error {
	return nil
}
