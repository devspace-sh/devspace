package testing

import (
	"context"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

// FakeConfigLoader is the fake config loader
type FakeConfigLoader struct {
	Config          *latest.Config
	GeneratedConfig *localcache.LocalCache
	Log             log.Logger
}

// NewFakeConfigLoader creates a new config loader
func NewFakeConfigLoader(generatedConfig *localcache.LocalCache, config *latest.Config, log log.Logger) loader.ConfigLoader {
	return &FakeConfigLoader{
		Config:          config,
		GeneratedConfig: generatedConfig,
		Log:             log,
	}
}

// Load implements interface
func (f *FakeConfigLoader) Load(ctx context.Context, client kubectl.Client, options *loader.ConfigOptions, log log.Logger) (config.Config, error) {
	if f.Config == nil {
		return nil, errors.New("Couldn't load config")
	}

	return config.NewConfig(nil, nil, f.Config, f.GeneratedConfig, nil, nil, constants.DefaultConfigPath), nil
}

func (f *FakeConfigLoader) LoadWithCache(ctx context.Context, localCache localcache.Cache, client kubectl.Client, options *loader.ConfigOptions, log log.Logger) (config.Config, error) {
	return nil, errors.New("Couldn't load config")
}

// ParseCommands implements interface
func (f *FakeConfigLoader) LoadWithParser(ctx context.Context, localCache localcache.Cache, client kubectl.Client, parser loader.Parser, options *loader.ConfigOptions, log log.Logger) (config.Config, error) {
	return nil, errors.New("Unsupported")
}

// LoadRaw implements interface
func (f *FakeConfigLoader) LoadRaw() (map[string]interface{}, error) {
	if f.Config == nil {
		return nil, errors.New("Couldn't load config")
	}

	out, err := yaml.Marshal(f.Config)
	if err != nil {
		return nil, err
	}

	retConfig := map[string]interface{}{}
	err = yaml.Unmarshal(out, &retConfig)
	if err != nil {
		return nil, err
	}

	return retConfig, nil
}

func (f *FakeConfigLoader) LoadLocalCache() (localcache.Cache, error) {
	return f.GeneratedConfig, nil
}

// Exists implements interface
func (f *FakeConfigLoader) Exists() bool {
	return f.Config != nil
}

func (f *FakeConfigLoader) ConfigPath() string {
	return ""
}

// SetDevSpaceRoot implements interface
func (f *FakeConfigLoader) SetDevSpaceRoot(log log.Logger) (bool, error) {
	return f.Config != nil, nil
}
