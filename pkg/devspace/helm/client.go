package helm

import (
	"github.com/loft-sh/devspace/pkg/devspace/helm/types"
	v4 "github.com/loft-sh/devspace/pkg/devspace/helm/v4"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// NewClient creates a new helm client based on the config
func NewClient(log log.Logger) (types.Client, error) {
	return v4.NewClient(log)
}
