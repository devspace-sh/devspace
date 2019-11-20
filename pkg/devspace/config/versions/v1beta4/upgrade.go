package v1beta4

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/config"
	next "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
)

// Upgrade upgrades the config
func (c *Config) Upgrade(log log.Logger) (config.Config, error) {
	nextConfig := &next.Config{}
	err := util.Convert(c, nextConfig)
	if err != nil {
		return nil, err
	}

	if len(c.Deployments) > 0 {
		for idx, deploymentConfig := range c.Deployments {
			if deploymentConfig.Helm != nil {
				nextConfig.Deployments[idx].Helm.V2 = true
				nextConfig.Deployments[idx].Helm.Atomic = ptr.ReverseBool(deploymentConfig.Helm.Rollback)
			}
		}
	}

	return nextConfig, nil
}

// UpgradeVarPaths upgrades the config
func (c *Config) UpgradeVarPaths(varPaths map[string]string, log log.Logger) error {
	return nil
}
