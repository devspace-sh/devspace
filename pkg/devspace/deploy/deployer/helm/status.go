package helm

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer"
	"github.com/loft-sh/devspace/pkg/devspace/helm"
)

// Status gets the status of the deployment
func (d *DeployConfig) Status() (*deployer.StatusResult, error) {
	var (
		deployTargetStr = d.getDeployTarget()
		err             error
	)

	if d.Helm == nil {
		// Get HelmClient
		d.Helm, err = helm.NewClient(d.config.Config(), d.DeploymentConfig, d.Kube, d.TillerNamespace, false, false, d.Log)
		if err != nil {
			return nil, err
		}
	}

	// Get all releases
	releases, err := d.Helm.ListReleases(d.DeploymentConfig.Helm)
	if err != nil {
		return &deployer.StatusResult{
			Name:   d.DeploymentConfig.Name,
			Type:   "Helm",
			Target: deployTargetStr,
			Status: fmt.Sprintf("Error: %v", err),
		}, nil
	}

	if releases == nil || len(releases) == 0 {
		return &deployer.StatusResult{
			Name:   d.DeploymentConfig.Name,
			Type:   "Helm",
			Target: deployTargetStr,
			Status: "Not deployed",
		}, nil
	}

	for _, release := range releases {
		if release.Name == d.DeploymentConfig.Name {
			if release.Status != "DEPLOYED" {
				return &deployer.StatusResult{
					Name:   d.DeploymentConfig.Name,
					Type:   "Helm",
					Target: deployTargetStr,
					Status: "Status:" + release.Status,
				}, nil
			}

			return &deployer.StatusResult{
				Name:   d.DeploymentConfig.Name,
				Type:   "Helm",
				Target: deployTargetStr,
				Status: "Deployed",
			}, nil
		}
	}

	return &deployer.StatusResult{
		Name:   d.DeploymentConfig.Name,
		Type:   "Helm",
		Target: deployTargetStr,
		Status: "Not deployed",
	}, nil
}

func (d *DeployConfig) getDeployTarget() string {
	if d.DeploymentConfig.Helm == nil || d.DeploymentConfig.Helm.Chart == nil {
		return "N/A"
	}

	retString := d.DeploymentConfig.Helm.Chart.Name
	if d.DeploymentConfig.Helm.Chart.Version != "" {
		retString += " (" + d.DeploymentConfig.Helm.Chart.Version + ")"
	}

	return retString
}
