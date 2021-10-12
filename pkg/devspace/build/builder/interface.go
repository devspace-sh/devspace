package builder

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// Interface defines methods for builders docker, kaniko and custom
type Interface interface {
	ShouldRebuild(cache *generated.CacheConfig, forceRebuild bool, log log.Logger) (bool, error)
	Build(devspacePID string, log log.Logger) error
}
