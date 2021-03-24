package config

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"gopkg.in/yaml.v2"
)

type Config interface {
	// Config returns the parsed config
	Config() *latest.Config

	// Raw returns the config as it was loaded from the devspace.yaml
	// including all sections
	Raw() map[interface{}]interface{}

	// Generated returns the generated config
	Generated() *generated.Config

	// Returns the profiles that can be parsed
	Profiles() ([]*latest.ProfileConfig, error)

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

func (c *config) Profiles() ([]*latest.ProfileConfig, error) {
	rawMap, err := CopyRaw(c.rawConfig)
	if err != nil {
		return nil, err
	}

	profiles, ok := rawMap["profiles"].([]interface{})
	if !ok {
		profiles = []interface{}{}
	}

	retProfiles := []*latest.ProfileConfig{}
	for _, profile := range profiles {
		profileMap, ok := profile.(map[interface{}]interface{})
		if !ok {
			continue
		}

		profileConfig := &latest.ProfileConfig{}
		o, err := yaml.Marshal(profileMap)
		if err != nil {
			continue
		}
		err = yaml.Unmarshal(o, profileConfig)
		if err != nil {
			continue
		}

		retProfiles = append(retProfiles, profileConfig)
	}

	return retProfiles, nil
}

func CopyRaw(in map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	o, err := yaml.Marshal(in)
	if err != nil {
		return nil, err
	}

	n := map[interface{}]interface{}{}
	err = yaml.Unmarshal(o, &n)
	if err != nil {
		return nil, err
	}

	return n, nil
}
