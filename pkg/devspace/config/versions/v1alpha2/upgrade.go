package v1alpha2

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1alpha3"
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
	if c.Images != nil {
		for key, image := range *c.Images {
			if image.Build != nil {
				if (*nextConfig.Images)[key].Build == nil {
					(*nextConfig.Images)[key].Build = &next.BuildConfig{}
				}

				if image.Build.DockerfilePath != nil {
					(*nextConfig.Images)[key].Build.Dockerfile = image.Build.DockerfilePath
				}
				if image.Build.ContextPath != nil {
					(*nextConfig.Images)[key].Build.Context = image.Build.ContextPath
				}
			}
		}
	}

	return nextConfig, nil
}
