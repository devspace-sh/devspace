package testing

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/log"
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
