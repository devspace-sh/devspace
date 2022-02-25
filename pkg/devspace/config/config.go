package config

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/remotecache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

type Config interface {
	RuntimeVariables

	// Config returns the parsed config
	Config() *latest.Config

	// Raw returns the config as it was loaded from the devspace.yaml
	// including all sections
	Raw() map[string]interface{}

	// Variables returns the variables that were resolved while
	// loading the config
	Variables() map[string]interface{}

	// LocalCache returns the local cache
	LocalCache() localcache.Cache

	// RemoteCache returns the remote cache
	RemoteCache() remotecache.Cache

	// Path returns the absolute path from which the config was loaded
	Path() string
}

func NewConfig(raw map[string]interface{}, parsed *latest.Config, localCache localcache.Cache, remoteCache remotecache.Cache, resolvedVariables map[string]interface{}, path string) Config {
	return &config{
		RuntimeVariables:  newRuntimeVariables(),
		rawConfig:         raw,
		parsedConfig:      parsed,
		localCache:        localCache,
		remoteCache:       remoteCache,
		resolvedVariables: resolvedVariables,
		path:              path,
	}
}

type config struct {
	RuntimeVariables

	rawConfig         map[string]interface{}
	parsedConfig      *latest.Config
	localCache        localcache.Cache
	remoteCache       remotecache.Cache
	resolvedVariables map[string]interface{}
	path              string
}

func (c *config) WithParsedConfig(conf *latest.Config) Config {
	n := *c
	n.parsedConfig = conf
	return &n
}

func (c *config) RemoteCache() remotecache.Cache {
	return c.remoteCache
}

func (c *config) Config() *latest.Config {
	return c.parsedConfig
}

func (c *config) Raw() map[string]interface{} {
	return c.rawConfig
}

func (c *config) LocalCache() localcache.Cache {
	return c.localCache
}

func (c *config) Variables() map[string]interface{} {
	return c.resolvedVariables
}

func (c *config) Path() string {
	return c.path
}

func Ensure(config Config) Config {
	retConfig := config
	if retConfig == nil {
		retConfig = NewConfig(nil, nil, nil, nil, nil, "")
	}
	if retConfig.Raw() == nil {
		retConfig = NewConfig(map[string]interface{}{}, retConfig.Config(), retConfig.LocalCache(), retConfig.RemoteCache(), retConfig.Variables(), retConfig.Path())
	}
	if retConfig.Config() == nil {
		retConfig = NewConfig(retConfig.Raw(), latest.NewRaw(), retConfig.LocalCache(), retConfig.RemoteCache(), retConfig.Variables(), retConfig.Path())
	}
	if retConfig.LocalCache() == nil {
		retConfig = NewConfig(retConfig.Raw(), retConfig.Config(), localcache.New(""), retConfig.RemoteCache(), retConfig.Variables(), retConfig.Path())
	}
	if retConfig.Variables() == nil {
		retConfig = NewConfig(retConfig.Raw(), retConfig.Config(), retConfig.LocalCache(), retConfig.RemoteCache(), map[string]interface{}{}, retConfig.Path())
	}

	return retConfig
}
