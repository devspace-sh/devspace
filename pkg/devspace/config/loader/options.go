package loader

import (
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"gopkg.in/yaml.v3"
)

// ConfigOptions defines options to load the config
type ConfigOptions struct {
	Dry bool

	OverrideName string

	// The profile that should be loaded
	Profiles []string
	// If the profile parents that are loaded from other sources should be refreshed
	ProfileRefresh bool
	// If the profile activations should be disabled
	DisableProfileActivation bool

	Vars []string
}

func (co *ConfigOptions) Clone() (*ConfigOptions, error) {
	out, err := yaml.Marshal(co)
	if err != nil {
		return nil, err
	}

	newCo := &ConfigOptions{}
	err = yamlutil.Unmarshal(out, newCo)
	if err != nil {
		return nil, err
	}

	return newCo, nil
}
