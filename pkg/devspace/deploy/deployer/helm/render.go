package helm

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"io"
)

// Render runs a `helm template`
func (d *DeployConfig) Render(ctx devspacecontext.Context, out io.Writer) error {
	_, deployValues, err := d.getDeploymentValues(ctx)
	if err != nil {
		return err
	}

	_, err = d.internalDeploy(ctx, deployValues, out)
	return err
}
