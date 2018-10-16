package helm

import (
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/devspace/helm"
	"k8s.io/client-go/kubernetes"
)

// DeployConfig holds the information necessary to deploy via helm
type DeployConfig struct {
	KubeClient       *kubernetes.Clientset
	TillerNamespace  string
	DeploymentConfig *v1.DeploymentConfig
}

// New creates a new helm deployment client
func New(kubectl *kubernetes.Clientset, deployConfig *v1.DeploymentConfig) (*DeployConfig, error) {
	config := configutil.GetConfig()
	return &DeployConfig{
		KubeClient:       kubectl,
		TillerNamespace:  *config.Tiller.Namespace,
		DeploymentConfig: deployConfig,
	}, nil
}

// Delete deletes the release
func (d *DeployConfig) Delete(verbose bool) error {
	// Delete with helm engine
	isDeployed := helm.IsTillerDeployed(d.KubeClient)
	if isDeployed == false {
		return nil
	}

	// Get HelmClient
	helmClient, err := helm.NewClient(d.KubeClient, false)
	if err != nil {
		return err
	}

	_, err = helmClient.DeleteRelease(*d.DeploymentConfig.Name, true)
	if err != nil {
		return err
	}

	return nil
}

// Status gets the status of the deployment
func (d *DeployConfig) Status() ([][]string, error) {
	var values [][]string

	// Get HelmClient
	helmClient, err := helm.NewClient(d.KubeClient, false)
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
