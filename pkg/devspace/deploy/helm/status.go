package helm

import "github.com/covexo/devspace/pkg/devspace/helm"

// Status gets the status of the deployment
func (d *DeployConfig) Status() ([][]string, error) {
	var values [][]string

	// Get HelmClient
	helmClient, err := helm.NewClient(d.TillerNamespace, d.Log, false)
	if err != nil {
		return nil, err
	}

	releases, err := helmClient.Client.ListReleases()
	if err != nil {
		values = append(values, []string{
			*d.DeploymentConfig.Name,
			"Error",
			*d.DeploymentConfig.Namespace,
			err.Error(),
		})

		return values, nil
	}

	if releases == nil || len(releases.Releases) == 0 {
		values = append(values, []string{
			*d.DeploymentConfig.Name,
			"Not Found",
			*d.DeploymentConfig.Namespace,
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
					*d.DeploymentConfig.Namespace,
					"HELM STATUS:" + release.Info.Status.Code.String(),
				})

				return values, nil
			}

			values = append(values, []string{
				*d.DeploymentConfig.Name,
				"Running",
				*d.DeploymentConfig.Namespace,
				"",
			})

			return values, nil
		}
	}

	values = append(values, []string{
		*d.DeploymentConfig.Name,
		"Not Found",
		*d.DeploymentConfig.Namespace,
		"No release found",
	})

	return values, nil
}
