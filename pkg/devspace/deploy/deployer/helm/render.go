package helm

import (
	"io"
)

// Render runs a `helm template`
func (d *DeployConfig) Render(builtImages map[string]string, out io.Writer) error {
	_, deployValues, err := d.getDeploymentValues(builtImages)
	if err != nil {
		return err
	}

	_, err = d.internalDeploy(deployValues, out)
	return err
}
