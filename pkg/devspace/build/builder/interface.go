package builder

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// Interface defines methods for builders docker, kaniko and custom
type Interface interface {
	ShouldRebuild(cache *generated.CacheConfig, ignoreContextPathChanges bool) (bool, error)
	Build(log log.Logger) error
}
