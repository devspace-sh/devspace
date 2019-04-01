package helm

import (
	"fmt"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/deploy"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm"
)

// Status gets the status of the deployment
func (d *DeployConfig) Status() (*deploy.StatusResult, error) {
	var (
		deployTargetStr = d.getDeployTarget()
	)

	// Get HelmClient
	helmClient, err := helm.NewClient(d.TillerNamespace, d.Log, false)
	if err != nil {
		return nil, err
	}

	// Get all releases
	releases, err := helmClient.Client.ListReleases()
	if err != nil {
		return &deploy.StatusResult{
			Name:   *d.DeploymentConfig.Name,
			Type:   "Helm",
			Target: deployTargetStr,
			Status: fmt.Sprintf("Error: %v", err),
		}, nil
	}

	if releases == nil || len(releases.Releases) == 0 {
		return &deploy.StatusResult{
			Name:   *d.DeploymentConfig.Name,
			Type:   "Helm",
			Target: deployTargetStr,
			Status: "Not deployed",
		}, nil
	}

	for _, release := range releases.Releases {
		if release.GetName() == *d.DeploymentConfig.Name {
			if release.Info.Status.Code.String() != "DEPLOYED" {
				return &deploy.StatusResult{
					Name:   *d.DeploymentConfig.Name,
					Type:   "Helm",
					Target: deployTargetStr,
					Status: "Status:" + release.Info.Status.Code.String(),
				}, nil
			}

			return &deploy.StatusResult{
				Name:   *d.DeploymentConfig.Name,
				Type:   "Helm",
				Target: deployTargetStr,
				Status: "Deployed " + time.Since(time.Unix(release.Info.LastDeployed.Seconds, 0)).String() + " ago",
			}, nil
		}
	}

	return &deploy.StatusResult{
		Name:   *d.DeploymentConfig.Name,
		Type:   "Helm",
		Target: deployTargetStr,
		Status: "Not deployed",
	}, nil
}

func (d *DeployConfig) getDeployTarget() string {
	if d.DeploymentConfig.Helm == nil || d.DeploymentConfig.Helm.Chart == nil {
		return "N/A"
	}

	retString := *d.DeploymentConfig.Helm.Chart.Name
	if d.DeploymentConfig.Helm.Chart.Version != nil {
		retString += " (" + *d.DeploymentConfig.Helm.Chart.Version + ")"
	}

	return retString
}
