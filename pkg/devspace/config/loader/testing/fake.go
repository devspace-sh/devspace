package testing

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/ghodss/yaml"
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

// New implements interface
func (f *FakeConfigLoader) New() *latest.Config {
	return f.Config
}

// Exists implements interface
func (f *FakeConfigLoader) Exists() bool {
	return f.Config != nil
}

// Load implements interface
func (f *FakeConfigLoader) Load() (*latest.Config, error) {
	if f.Config == nil {
		return nil, errors.New("Couldn't load config")
	}

	return f.Config, nil
}

// LoadFromPath implements interface
func (f *FakeConfigLoader) LoadFromPath(generatedConfig *generated.Config, path string) (*latest.Config, error) {
	if f.Config == nil {
		return nil, errors.New("Couldn't load config")
	}

	return f.Config, nil
}

// LoadRaw implements interface
func (f *FakeConfigLoader) LoadRaw(path string) (map[interface{}]interface{}, error) {
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

// LoadWithoutProfile implements interface
func (f *FakeConfigLoader) LoadWithoutProfile() (*latest.Config, error) {
	if f.Config == nil {
		return nil, errors.New("Couldn't load config")
	}

	return f.Config, nil
}

// ParseCommands implements interface
func (f *FakeConfigLoader) ParseCommands(generatedConfig *generated.Config, data map[interface{}]interface{}) ([]*latest.CommandConfig, error) {
	return loader.NewConfigLoader(nil, f.Log).ParseCommands(generatedConfig, data)
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

// RestoreVars implements interface
func (f *FakeConfigLoader) RestoreVars(config *latest.Config) (*latest.Config, error) {
	// Cloned config
	clonedConfig := &latest.Config{}

	// Copy config
	err := util.Convert(config, clonedConfig)
	if err != nil {
		return nil, errors.Wrap(err, "convert cloned config")
	}

	return clonedConfig, nil
}

// SetDevSpaceRoot implements interface
func (f *FakeConfigLoader) SetDevSpaceRoot() (bool, error) {
	return f.Config != nil, nil
}
