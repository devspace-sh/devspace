package config

// Config is the interface for each config version
type Config interface {
	GetVersion() string
	Upgrade() (Config, error)
	UpgradeVarPaths(varPaths map[string]string) error
}

// New creates a new config
type New func() Config
