package v1beta8

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

	// convert image configs
	for key, value := range c.Images {
		if value == nil {
			continue
		}

		nextConfig.Images[key].PreferSyncOverRebuild = true
	}

	return nextConfig, nil
}

// UpgradeVarPaths upgrades the config
func (c *Config) UpgradeVarPaths(varPaths map[string]string, log log.Logger) error {
	return nil
}
