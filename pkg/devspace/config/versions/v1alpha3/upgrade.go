package v1alpha3

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1alpha4"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// Upgrade upgrades the config
func (c *Config) Upgrade(log log.Logger) (config.Config, error) {
	nextConfig := &next.Config{}
	err := util.Convert(c, nextConfig)
	if err != nil {
		return nil, err
	}

	// Convert dockerfilepath and contextpath
	if c.Deployments != nil {
		for key, deployConfig := range *c.Deployments {
			if deployConfig.Helm != nil {
				if (*nextConfig.Deployments)[key].Helm == nil {
					(*nextConfig.Deployments)[key].Helm = &next.HelmConfig{}
				}
				if deployConfig.Helm.ChartPath != nil {
					(*nextConfig.Deployments)[key].Helm.Chart = &next.ChartConfig{
						Name: deployConfig.Helm.ChartPath,
					}
				}
				if deployConfig.Helm.Overrides != nil {
					(*nextConfig.Deployments)[key].Helm.ValuesFiles = deployConfig.Helm.Overrides
				}
				if deployConfig.Helm.OverrideValues != nil {
					(*nextConfig.Deployments)[key].Helm.Values = deployConfig.Helm.OverrideValues
				}
			}
		}
	}

	return nextConfig, nil
}
