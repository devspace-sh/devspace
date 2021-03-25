package v1beta5

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta6"
	"github.com/loft-sh/devspace/pkg/util/log"
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
			if config == nil {
				continue
			}

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
