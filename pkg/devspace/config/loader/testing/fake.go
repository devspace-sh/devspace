package testing

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

// FakeConfigLoader is the fake config loader
type FakeConfigLoader struct {
	Config          *latest.Config
	GeneratedConfig *generated.Config
	Log             log.Logger
}

// NewFakeConfigLoader creates a new config loader
func NewFakeConfigLoader(generatedConfig *generated.Config, config *latest.Config, log log.Logger) loader.ConfigLoader {
	return &FakeConfigLoader{
		Config:          config,
		GeneratedConfig: generatedConfig,
		Log:             log,
	}
}

// Exists implements interface
func (f *FakeConfigLoader) Exists() bool {
	return f.Config != nil
}

// Load implements interface
func (f *FakeConfigLoader) Load(options *loader.ConfigOptions, log log.Logger) (config.Config, error) {
	if f.Config == nil {
		return nil, errors.New("Couldn't load config")
	}

	return config.NewConfig(nil, f.Config, f.GeneratedConfig, nil), nil
}

func (f *FakeConfigLoader) ConfigPath() string {
	return ""
}

// LoadFromPath implements interface
func (f *FakeConfigLoader) LoadFromPath(generatedConfig *generated.Config, path string) (*latest.Config, error) {
	if f.Config == nil {
		return nil, errors.New("Couldn't load config")
	}

	return f.Config, nil
}

func (f *FakeConfigLoader) RestoreLoadSave(client kubectl.Client) (*latest.Config, error) {
	return f.Config, nil
}

func (f *FakeConfigLoader) LoadGenerated(options *loader.ConfigOptions) (*generated.Config, error) {
	return f.GeneratedConfig, nil
}

// LoadRaw implements interface
func (f *FakeConfigLoader) LoadRaw() (map[interface{}]interface{}, error) {
	if f.Config == nil {
		return nil, errors.New("Couldn't load config")
	}

	out, err := yaml.Marshal(f.Config)
	if err != nil {
		return nil, err
	}

	retConfig := map[interface{}]interface{}{}
	err = yaml.Unmarshal(out, &retConfig)
	if err != nil {
		return nil, err
	}

	return retConfig, nil
}

// GetProfiles implements interface
func (f *FakeConfigLoader) LoadProfiles() ([]*latest.ProfileConfig, error) {
	return f.Config.Profiles, nil
}

// ParseCommands implements interface
func (f *FakeConfigLoader) LoadWithParser(parser loader.Parser, options *loader.ConfigOptions, log log.Logger) (config.Config, error) {
	return nil, fmt.Errorf("Unsupported")
}

// Generated implements interface
func (f *FakeConfigLoader) Generated() (*generated.Config, error) {
	if f.GeneratedConfig == nil {
		return nil, errors.New("Couldn't load config")
	}

	return f.GeneratedConfig, nil
}

// SaveGenerated implements interface
func (f *FakeConfigLoader) SaveGenerated(generatedConfig *generated.Config) error {
	return nil
}

// Save implements interface
func (f *FakeConfigLoader) Save(config *latest.Config) error {
	return nil
}

// SetDevSpaceRoot implements interface
func (f *FakeConfigLoader) SetDevSpaceRoot(log log.Logger) (bool, error) {
	return f.Config != nil, nil
}
