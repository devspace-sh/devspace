package latest

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/config"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// Upgrade upgrades the config
func (c *Config) Upgrade(log log.Logger) (config.Config, error) {
	panic("unimplemented")
}

// UpgradeVarPaths upgrades the config
func (c *Config) UpgradeVarPaths(varPaths map[string]string, log log.Logger) error {
	panic("unimplemented")
}
