package helm

import (
	"io"
)

// Render runs a `helm template`
func (d *DeployConfig) Render(builtImages map[string]string, out io.Writer) error {
	_, _, err := d.internalDeploy(true, builtImages, out)
	return err
}
