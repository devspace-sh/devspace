package config

import "github.com/loft-sh/devspace/pkg/util/log"

// Config is the interface for each config version
type Config interface {
	GetVersion() string
	Upgrade(log log.Logger) (Config, error)
	UpgradeVarPaths(varPaths map[string]string, log log.Logger) error
}

// New creates a new config
type New func() Config
