package helm

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	v1 "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

// DeployConfig holds the information necessary to deploy via helm
type DeployConfig struct {
	// Public because we can switch them to fake clients for testing
	Kube kubernetes.Interface
	Helm helm.Interface

	TillerNamespace  string
	DeploymentConfig *v1.DeploymentConfig
	Log              log.Logger
}

// New creates a new helm deployment client
func New(kubectl kubernetes.Interface, deployConfig *v1.DeploymentConfig, log log.Logger) (*DeployConfig, error) {
	config := configutil.GetConfig()
	tillerNamespace, err := configutil.GetDefaultNamespace(config)
	if err != nil {
		return nil, err
	}
	if deployConfig.Helm.TillerNamespace != nil && *deployConfig.Helm.TillerNamespace != "" {
		tillerNamespace = *deployConfig.Helm.TillerNamespace
	}

	return &DeployConfig{
		Kube:             kubectl,
		TillerNamespace:  tillerNamespace,
		DeploymentConfig: deployConfig,
		Log:              log,
	}, nil
}

// Delete deletes the release
func (d *DeployConfig) Delete(cache *generated.CacheConfig) error {
	// Delete with helm engine
	isDeployed := helm.IsTillerDeployed(d.Kube, d.TillerNamespace)
	if isDeployed == false {
		return nil
	}

	if d.Helm == nil {
		var err error

		// Get HelmClient
		d.Helm, err = helm.NewClient(d.TillerNamespace, d.Log, false)
		if err != nil {
			return errors.Wrap(err, "new helm client")
		}
	}

	_, err := d.Helm.DeleteRelease(*d.DeploymentConfig.Name, true)
	if err != nil {
		return err
	}

	// Delete from cache
	delete(cache.Deployments, *d.DeploymentConfig.Name)
	return nil
}
