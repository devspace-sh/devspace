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

	// Returns the variables that were resolved while
	// loading the config
	Variables() map[string]interface{}
}

func NewConfig(raw map[interface{}]interface{}, parsed *latest.Config, generatedConfig *generated.Config, resolvedVariables map[string]interface{}) Config {
	return &config{
		rawConfig:         raw,
		parsedConfig:      parsed,
		generatedConfig:   generatedConfig,
		resolvedVariables: resolvedVariables,
	}
}

type config struct {
	rawConfig         map[interface{}]interface{}
	parsedConfig      *latest.Config
	generatedConfig   *generated.Config
	resolvedVariables map[string]interface{}
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
