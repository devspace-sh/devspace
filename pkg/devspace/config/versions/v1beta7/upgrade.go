package v1beta7

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta8"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// Upgrade upgrades the config
func (c *Config) Upgrade(log log.Logger) (config.Config, error) {
	nextConfig := &next.Config{}
	err := util.Convert(c, nextConfig)
	if err != nil {
		return nil, err
	}

	// Kaniko: Flags -> Args
	// Kubectl: Flags -> ApplyArgs

	// Convert image configs
	for key, value := range c.Images {
		if value == nil {
			continue
		}

		if value.Build != nil {
			if value.Build.Kaniko != nil && len(value.Build.Kaniko.Flags) > 0 {
				if nextConfig.Images[key].Build == nil {
					nextConfig.Images[key].Build = &next.BuildConfig{}
				}
				if nextConfig.Images[key].Build.Kaniko == nil {
					nextConfig.Images[key].Build.Kaniko = &next.KanikoConfig{}
				}
				nextConfig.Images[key].Build.Kaniko.Args = value.Build.Kaniko.Flags
			}
		}
	}

	// Convert deployment configs
	for idx, value := range c.Deployments {
		if value == nil {
			continue
		}

		if value.Kubectl != nil && len(value.Kubectl.Flags) > 0 {
			nextConfig.Deployments[idx].Kubectl.ApplyArgs = value.Kubectl.Flags
		}
	}

	return nextConfig, nil
}
