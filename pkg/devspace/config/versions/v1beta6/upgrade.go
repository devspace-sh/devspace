package v1beta6

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta7"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// Upgrade upgrades the config
func (c *Config) Upgrade(log log.Logger) (config.Config, error) {
	nextConfig := &next.Config{}
	err := util.Convert(c, nextConfig)
	if err != nil {
		return nil, err
	}

	if c.Dev != nil && len(c.Dev.Sync) > 0 {
		for key, config := range c.Dev.Sync {
			if config == nil {
				continue
			}

			if config.DownloadOnInitialSync != nil && *config.DownloadOnInitialSync {
				nextConfig.Dev.Sync[key].InitialSync = next.InitialSyncStrategyPreferLocal
			}
		}
	}

	for key, value := range c.Images {
		if value == nil {
			continue
		}

		if value.Tag != "" {
			nextConfig.Images[key].Tags = []string{value.Tag}
		}
	}

	return nextConfig, nil
}
