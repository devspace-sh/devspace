package helm

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/v2"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/v3"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// NewClient creates a new helm client based on the config
func NewClient(config *latest.Config, deployConfig *latest.DeploymentConfig, kubeClient kubectl.Client, tillerNamespace string, upgradeTiller bool, log log.Logger) (types.Client, error) {
	if deployConfig.Helm.V2 == true {
		return v2.NewClient(config, kubeClient, tillerNamespace, log, upgradeTiller)
	}

	return v3.NewClient(kubeClient, deployConfig.Helm.Driver, log)
}
