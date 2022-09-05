package helm

import (
	"io"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/pkg/errors"
)

// Render runs a `helm template`
func (d *DeployConfig) Render(ctx devspacecontext.Context, out io.Writer) error {
	_, deployValues, err := d.getDeploymentValues(ctx)
	if err != nil {
		return err
	}

	if d.DeploymentConfig.Helm.Chart.Source != nil {
		_, err := d.Helm.DownloadChart(ctx, d.DeploymentConfig.Helm)
		if err != nil {
			return errors.Wrap(err, "download chart")
		}
	}

	_, err = d.internalDeploy(ctx, deployValues, out)
	return err
}
