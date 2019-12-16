package v1beta5

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/config"
	next "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// Upgrade upgrades the config
func (c *Config) Upgrade(log log.Logger) (config.Config, error) {
	nextConfig := &next.Config{}
	err := util.Convert(c, nextConfig)
	if err != nil {
		return nil, err
	}

	if len(c.Images) > 0 {
		for key, config := range c.Images {
			if config.Build != nil && config.Build.Custom != nil && len(config.Build.Custom.Args) > 0 {
				if nextConfig.Images[key].Build == nil {
					nextConfig.Images[key].Build = &next.BuildConfig{}
				}
				if nextConfig.Images[key].Build.Custom == nil {
					nextConfig.Images[key].Build.Custom = &next.CustomConfig{}
				}

				nextConfig.Images[key].Build.Custom.Args = config.Build.Custom.Args
			}
		}
	}

	return nextConfig, nil
}

// UpgradeVarPaths upgrades the config
func (c *Config) UpgradeVarPaths(varPaths map[string]string, log log.Logger) error {
	return nil
}
