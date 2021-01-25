package helm

import (
	"io"

	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
)

// Render runs a `helm template`
func (d *DeployConfig) Render(cache *generated.CacheConfig, builtImages map[string]string, out io.Writer) error {
	_, _, err := d.internalDeploy(cache, true, builtImages, out)
	return err
}
