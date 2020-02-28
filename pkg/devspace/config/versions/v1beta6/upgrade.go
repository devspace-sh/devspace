package v1beta6

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
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

	if c.Dev != nil && len(c.Dev.Sync) > 0 {
		for key, config := range c.Dev.Sync {
			if config == nil {
				continue
			}

			if config.DownloadOnInitialSync != nil && *config.DownloadOnInitialSync == true {
				nextConfig.Dev.Sync[key].InitialSync = latest.InitialSyncStrategyPreferLocal
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

// UpgradeVarPaths upgrades the config
func (c *Config) UpgradeVarPaths(varPaths map[string]string, log log.Logger) error {
	return nil
}
