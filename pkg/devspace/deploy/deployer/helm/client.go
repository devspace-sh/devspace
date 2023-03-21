package helm

import (
	"fmt"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"os"
	"path/filepath"

	"github.com/loft-sh/devspace/assets"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer"
	"github.com/loft-sh/devspace/pkg/devspace/helm"
	helmtypes "github.com/loft-sh/devspace/pkg/devspace/helm/types"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

// ComponentChartFolder holds the component charts
const ComponentChartFolder = "component-chart"

// DevSpaceChartConfig is the config that holds the devspace chart information
var DevSpaceChartConfig = &latest.ChartConfig{
	Name:    "component-chart",
	Version: "0.8.6",
	RepoURL: "https://charts.devspace.sh",
}

// DeployConfig holds the information necessary to deploy via helm
type DeployConfig struct {
	// Public because we can switch them to fake clients for testing
	Helm             helmtypes.Client
	DeploymentConfig *latest.DeploymentConfig
}

// New creates a new helm deployment client
func New(helmClient helmtypes.Client, deployConfig *latest.DeploymentConfig) (deployer.Interface, error) {
	// Exchange chart
	if deployConfig.Helm.Chart == nil || (deployConfig.Helm.Chart.Name == DevSpaceChartConfig.Name && deployConfig.Helm.Chart.RepoURL == DevSpaceChartConfig.RepoURL) {
		// extract component chart if possible
		filename := "component-chart-" + DevSpaceChartConfig.Version + ".tgz"
		componentChartBytes, err := assets.Asset(filename)
		if err == nil {
			homedir, _ := homedir.Dir()
			completePath := filepath.Join(homedir, constants.DefaultHomeDevSpaceFolder, ComponentChartFolder, filename)
			_, err := os.Stat(completePath)
			if err != nil {
				// make folder
				err = os.MkdirAll(filepath.Dir(completePath), 0755)
				if err != nil {
					return nil, err
				}

				// write file
				err = os.WriteFile(completePath, componentChartBytes, 0666)
				if err != nil {
					return nil, fmt.Errorf("error writing component chart to file: %v", err)
				}
			}

			deployConfig.Helm.Chart = &latest.ChartConfig{
				Name: completePath,
			}
		} else {
			deployConfig.Helm.Chart = DevSpaceChartConfig
		}
	}

	return &DeployConfig{
		Helm:             helmClient,
		DeploymentConfig: deployConfig,
	}, nil
}

// Delete deletes the deployment
func Delete(ctx devspacecontext.Context, deploymentName string) error {
	deploymentCache, ok := ctx.Config().RemoteCache().GetDeployment(deploymentName)
	if !ok || deploymentCache.Helm == nil || deploymentCache.Helm.Release == "" || deploymentCache.Helm.ReleaseNamespace == "" {
		return nil
	}

	helmClient, err := helm.NewClient(ctx.Log())
	if err != nil {
		return errors.Wrap(err, "new helm client")
	}

	err = helmClient.DeleteRelease(ctx, deploymentCache.Helm.Release, deploymentCache.Helm.ReleaseNamespace)
	if err != nil {
		return err
	}

	return nil
}
