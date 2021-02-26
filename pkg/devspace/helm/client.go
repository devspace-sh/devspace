package helm

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/helm/types"
	"github.com/loft-sh/devspace/pkg/devspace/helm/v2"
	v3 "github.com/loft-sh/devspace/pkg/devspace/helm/v3"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// NewClient creates a new helm client based on the config
func NewClient(config *latest.Config, deployConfig *latest.DeploymentConfig, kubeClient kubectl.Client, tillerNamespace string, upgradeTiller, dryInit bool, log log.Logger) (types.Client, error) {
	if deployConfig.Helm.V2 == true {
		return v2.NewClient(config, kubeClient, tillerNamespace, log)
	}

	return v3.NewClient(kubeClient, log)
}
