package loader

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"gopkg.in/yaml.v2"
)

func OptionsWithGeneratedConfig(generatedConfig *generated.Config) *ConfigOptions {
	return &ConfigOptions{
		GeneratedConfig: generatedConfig,
	}
}

// ConfigOptions defines options to load the config
type ConfigOptions struct {
	// KubeClient is needed if variables were saved in the namespace
	KubeClient kubectl.Client `yaml:"-" json:"-"`

	// Optionally passed generated config that is used for loading the config
	GeneratedConfig *generated.Config

	KubeContext string
	Namespace   string

	// If the config is loaded from a dependency, this points to the original
	// path where the base config was loaded from
	BasePath string
	// The profile that should be loaded
	Profile string
	// If specified profiles that should be loaded before the actual profile
	ProfileParents []string
	// If the profile parents that are loaded from other sources should be refreshed
	ProfileRefresh bool

	Vars []string

	RestoreVars    bool
	SaveVars       bool
	VarsSecretName string

	// can be used for testing
	generatedLoader generated.ConfigLoader `yaml:"-" json:"-"`
}

// Clone clones the config options
func (co *ConfigOptions) Clone() (*ConfigOptions, error) {
	out, err := yaml.Marshal(co)
	if err != nil {
		return nil, err
	}

	newCo := &ConfigOptions{}
	err = yaml.Unmarshal(out, newCo)
	if err != nil {
		return nil, err
	}

	return newCo, nil
}
