package v1beta4

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta5"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"
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
