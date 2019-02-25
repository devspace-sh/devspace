package helm

import (
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm"
)

// Status gets the status of the deployment
func (d *DeployConfig) Status() ([][]string, error) {
	var values [][]string
	config := configutil.GetConfig()

	// Get HelmClient
	helmClient, err := helm.NewClient(d.TillerNamespace, d.Log, false)
	if err != nil {
		return nil, err
	}

	namespace, err := configutil.GetDefaultNamespace(config)
	if err != nil {
		return nil, err
	}
	if d.DeploymentConfig.Namespace != nil {
		namespace = *d.DeploymentConfig.Namespace
	}

	releases, err := helmClient.Client.ListReleases()
	if err != nil {
		values = append(values, []string{
			*d.DeploymentConfig.Name,
			"Error",
			namespace,
			err.Error(),
		})

		return values, nil
	}

	if releases == nil || len(releases.Releases) == 0 {
		values = append(values, []string{
			*d.DeploymentConfig.Name,
			"Not Found",
			namespace,
			"No release found",
		})

		return values, nil
	}

	for _, release := range releases.Releases {
		if release.GetName() == *d.DeploymentConfig.Name {
			if release.Info.Status.Code.String() != "DEPLOYED" {
				values = append(values, []string{
					*d.DeploymentConfig.Name,
					"Error",
					namespace,
					"HELM STATUS:" + release.Info.Status.Code.String(),
				})

				return values, nil
			}

			values = append(values, []string{
				*d.DeploymentConfig.Name,
				"Deployed",
				namespace,
				"Deployed: " + time.Unix(release.Info.LastDeployed.Seconds, 0).String(),
			})

			return values, nil
		}
	}

	values = append(values, []string{
		*d.DeploymentConfig.Name,
		"Not Found",
		namespace,
		"No release found",
	})

	return values, nil
}
