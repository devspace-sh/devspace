package config

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

type Config interface {
	// Config returns the parsed config
	Config() *latest.Config

	// Raw returns the config as it was loaded from the devspace.yaml
	// including all sections
	Raw() map[interface{}]interface{}

	// Generated returns the generated config
	Generated() *generated.Config

	// Variables returns the variables that were resolved while
	// loading the config
	Variables() map[string]interface{}

	// Path returns the absolute path from which the config was loaded
	Path() string
}

func NewConfig(raw map[interface{}]interface{}, parsed *latest.Config, generatedConfig *generated.Config, resolvedVariables map[string]interface{}, path string) Config {
	return &config{
		rawConfig:         raw,
		parsedConfig:      parsed,
		generatedConfig:   generatedConfig,
		resolvedVariables: resolvedVariables,
		path:              path,
	}
}

type config struct {
	rawConfig         map[interface{}]interface{}
	parsedConfig      *latest.Config
	generatedConfig   *generated.Config
	resolvedVariables map[string]interface{}
	path              string
}

func (c *config) Config() *latest.Config {
	return c.parsedConfig
}

func (c *config) Raw() map[interface{}]interface{} {
	return c.rawConfig
}

func (c *config) Generated() *generated.Config {
	return c.generatedConfig
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
		retConfig = NewConfig(nil, nil, nil, nil, "")
	}
	if retConfig.Raw() == nil {
		retConfig = NewConfig(map[interface{}]interface{}{}, retConfig.Config(), retConfig.Generated(), retConfig.Variables(), retConfig.Path())
	}
	if retConfig.Config() == nil {
		retConfig = NewConfig(retConfig.Raw(), latest.NewRaw(), retConfig.Generated(), retConfig.Variables(), retConfig.Path())
	}
	if retConfig.Generated() == nil {
		retConfig = NewConfig(retConfig.Raw(), retConfig.Config(), generated.New(), retConfig.Variables(), retConfig.Path())
	}
	if retConfig.Variables() == nil {
		retConfig = NewConfig(retConfig.Raw(), retConfig.Config(), retConfig.Generated(), map[string]interface{}{}, retConfig.Path())
	}

	return retConfig
}
