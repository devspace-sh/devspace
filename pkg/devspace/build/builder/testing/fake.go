package testing

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// Builder is a fake implementation of the Interface interface
type Builder struct {
}

// ShouldRebuild is a fake implementation of the function
func (b *Builder) ShouldRebuild(cache *generated.CacheConfig, forceRebuild, ignoreContextPathChanges bool) (bool, error) {
	return forceRebuild, nil
}

// Build is a fake implementation of the function
func (b *Builder) Build(log log.Logger) error {
	return nil
}
