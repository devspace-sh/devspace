package helm

import (
	"github.com/loft-sh/devspace/pkg/devspace/helm/types"
	v3 "github.com/loft-sh/devspace/pkg/devspace/helm/v3"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// NewClient creates a new helm client based on the config
func NewClient(log log.Logger) (types.Client, error) {
	return v3.NewClient(log)
}
