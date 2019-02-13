package helm

import (
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	v1 "github.com/covexo/devspace/pkg/devspace/config/versions/latest"
	"github.com/covexo/devspace/pkg/devspace/helm"
	"github.com/covexo/devspace/pkg/util/log"
	"k8s.io/client-go/kubernetes"
)

// DeployConfig holds the information necessary to deploy via helm
type DeployConfig struct {
	KubeClient       *kubernetes.Clientset
	TillerNamespace  string
	DeploymentConfig *v1.DeploymentConfig
	Log              log.Logger
}

// New creates a new helm deployment client
func New(kubectl *kubernetes.Clientset, deployConfig *v1.DeploymentConfig, log log.Logger) (*DeployConfig, error) {
	config := configutil.GetConfig()
	tillerNamespace, err := configutil.GetDefaultNamespace(config)
	if err != nil {
		return nil, err
	}
	if deployConfig.Helm.TillerNamespace != nil && *deployConfig.Helm.TillerNamespace != "" {
		tillerNamespace = *deployConfig.Helm.TillerNamespace
	}

	return &DeployConfig{
		KubeClient:       kubectl,
		TillerNamespace:  tillerNamespace,
		DeploymentConfig: deployConfig,
		Log:              log,
	}, nil
}

// Delete deletes the release
func (d *DeployConfig) Delete() error {
	// Delete with helm engine
	isDeployed := helm.IsTillerDeployed(d.KubeClient, d.TillerNamespace)
	if isDeployed == false {
		return nil
	}

	// Get HelmClient
	helmClient, err := helm.NewClient(d.TillerNamespace, d.Log, false)
	if err != nil {
		return err
	}

	_, err = helmClient.DeleteRelease(*d.DeploymentConfig.Name, true)
	if err != nil {
		return err
	}

	return nil
}
