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

	// RawBeforeConversion returns the config right before it was converted
	// up to the latest version
	RawBeforeConversion() map[string]interface{}

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

func NewConfig(raw map[string]interface{}, rawBeforeConversion map[string]interface{}, parsed *latest.Config, localCache localcache.Cache, remoteCache remotecache.Cache, resolvedVariables map[string]interface{}, path string) Config {
	return &config{
		RuntimeVariables:    newRuntimeVariables(),
		rawConfig:           raw,
		rawBeforeConversion: rawBeforeConversion,
		parsedConfig:        parsed,
		localCache:          localCache,
		remoteCache:         remoteCache,
		resolvedVariables:   resolvedVariables,
		path:                path,
	}
}

type config struct {
	RuntimeVariables

	rawConfig           map[string]interface{}
	rawBeforeConversion map[string]interface{}
	parsedConfig        *latest.Config
	localCache          localcache.Cache
	remoteCache         remotecache.Cache
	resolvedVariables   map[string]interface{}
	path                string
}

func (c *config) RawBeforeConversion() map[string]interface{} {
	return c.rawBeforeConversion
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
	newVariables := map[string]interface{}{}
	for k, v := range c.resolvedVariables {
		newVariables[k] = v
	}

	return newVariables
}

func (c *config) Path() string {
	return c.path
}

func Ensure(config Config) Config {
	retConfig := config
	if retConfig == nil {
		retConfig = NewConfig(nil, nil, nil, nil, nil, nil, "")
	}
	if retConfig.Raw() == nil {
		retConfig = NewConfig(map[string]interface{}{}, retConfig.RawBeforeConversion(), retConfig.Config(), retConfig.LocalCache(), retConfig.RemoteCache(), retConfig.Variables(), retConfig.Path())
	}
	if retConfig.RawBeforeConversion() == nil {
		retConfig = NewConfig(retConfig.Raw(), map[string]interface{}{}, retConfig.Config(), retConfig.LocalCache(), retConfig.RemoteCache(), retConfig.Variables(), retConfig.Path())
	}
	if retConfig.Config() == nil {
		retConfig = NewConfig(retConfig.Raw(), retConfig.RawBeforeConversion(), latest.NewRaw(), retConfig.LocalCache(), retConfig.RemoteCache(), retConfig.Variables(), retConfig.Path())
	}
	if retConfig.LocalCache() == nil {
		retConfig = NewConfig(retConfig.Raw(), retConfig.RawBeforeConversion(), retConfig.Config(), localcache.New(""), retConfig.RemoteCache(), retConfig.Variables(), retConfig.Path())
	}
	if retConfig.Variables() == nil {
		retConfig = NewConfig(retConfig.Raw(), retConfig.RawBeforeConversion(), retConfig.Config(), retConfig.LocalCache(), retConfig.RemoteCache(), map[string]interface{}{}, retConfig.Path())
	}

	if config != nil {
		runtimeVars := config.ListRuntimeVariables()
		for k, v := range runtimeVars {
			retConfig.SetRuntimeVariable(k, v)
		}
	}

	return retConfig
}
