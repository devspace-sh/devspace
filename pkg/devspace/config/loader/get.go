package loader

import (
	"io/ioutil"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
)

// ConfigLoader is the base interface for the main config loader
type ConfigLoader interface {
	New() *latest.Config
	Exists() bool

	Load() (*latest.Config, error)
	LoadRaw() (map[interface{}]interface{}, error)
	LoadWithoutProfile() (*latest.Config, error)

	ConfigPath() string
	GetProfiles() ([]string, error)
	ParseCommands() ([]*latest.CommandConfig, error)

	ResolvedVars() map[string]string
	Generated() (*generated.Config, error)
	SaveGenerated() error

	RestoreVars(config *latest.Config) (*latest.Config, error)
	Save(config *latest.Config) error
	SetDevSpaceRoot() (bool, error)
}

type configLoader struct {
	generatedLoader generated.ConfigLoader
	generatedConfig *generated.Config

	kubeConfigLoader kubeconfig.Loader

	resolvedVars map[string]string
	options      *ConfigOptions
	log          log.Logger
}

// NewConfigLoader creates a new config loader with the given options
func NewConfigLoader(options *ConfigOptions, log log.Logger) ConfigLoader {
	if options == nil {
		options = &ConfigOptions{}
	}

	// Set loaded vars for this
	options.LoadedVars = make(map[string]string)

	return &configLoader{
		generatedConfig:  options.GeneratedConfig,
		generatedLoader:  generated.NewConfigLoader(options.Profile),
		kubeConfigLoader: kubeconfig.NewLoader(),
		options:          options,
		log:              log,
	}
}

// LoadGenerated loads the generated config
func (l *configLoader) Generated() (*generated.Config, error) {
	var err error
	if l.generatedConfig == nil {
		l.generatedConfig, err = l.generatedLoader.Load()
	}

	return l.generatedConfig, err
}

// SaveGenerated is a convenience method to save the generated config
func (l *configLoader) SaveGenerated() error {
	if l.generatedLoader == nil {
		return nil
	}

	generatedConfig, err := l.Generated()
	if err != nil {
		return err
	}

	return l.generatedLoader.Save(generatedConfig)
}

func (l *configLoader) ResolvedVars() map[string]string {
	return l.resolvedVars
}

// Exists checks whether the yaml file for the config exists or the configs.yaml exists
func (l *configLoader) Exists() bool {
	path := l.ConfigPath()

	return configExistsInPath(path)
}

// configExistsInPath checks wheter a devspace configuration exists at a certain path
func configExistsInPath(path string) bool {
	// Check devspace.yaml
	_, err := os.Stat(path)
	if err == nil {
		return true
	}

	return false // Normal config file found
}

// New initializes a new config object
func (l *configLoader) New() *latest.Config {
	return latest.New().(*latest.Config)
}

// ConfigOptions defines options to load the config
type ConfigOptions struct {
	KubeContext string
	Namespace   string

	// The path the config should be loaded from (e.g. /test/devspace.yaml)
	ConfigPath string
	// If the config is loaded from a dependency, this points to the original
	// path where the base config was loaded from
	BasePath string
	// The profile that should be loaded
	Profile string
	// If the profile parents that are loaded from other sources should be refreshed
	ProfileRefresh bool

	GeneratedConfig *generated.Config
	LoadedVars      map[string]string
	Vars            []string
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

// GetBaseConfig returns the config
func (l *configLoader) LoadWithoutProfile() (*latest.Config, error) {
	return l.loadInternal(false)
}

// GetConfig returns the config merged with all potential overwrite files
func (l *configLoader) Load() (*latest.Config, error) {
	return l.loadInternal(true)
}

// GetRawConfig loads the raw config from a given path
func (l *configLoader) LoadRaw() (map[interface{}]interface{}, error) {
	// What path should we use
	configPath := l.ConfigPath()
	_, err := os.Stat(configPath)
	if err != nil {
		return nil, errors.Errorf("Couldn't load '%s': %v", configPath, err)
	}

	fileContent, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	rawMap := map[interface{}]interface{}{}
	err = yaml.Unmarshal(fileContent, &rawMap)
	if err != nil {
		return nil, err
	}

	return rawMap, nil
}

func (l *configLoader) ConfigPath() string {
	path := constants.DefaultConfigPath
	if l.options.ConfigPath != "" {
		path = l.options.ConfigPath
	}

	return path
}

// loadInternal loads the config internally
func (l *configLoader) loadInternal(allowProfile bool) (*latest.Config, error) {
	// Get generated config
	generatedConfig, err := l.Generated()
	if err != nil {
		return nil, err
	}

	// Check if we should load a specific config
	if allowProfile && generatedConfig.ActiveProfile != "" && l.options.Profile == "" {
		l.options.Profile = generatedConfig.ActiveProfile
	} else if !allowProfile {
		l.options.Profile = ""
	}

	// Load the raw config
	rawMap, err := l.LoadRaw()
	if err != nil {
		return nil, err
	}

	// Parse the config
	config, err := l.parseConfig(rawMap)
	if err != nil {
		return nil, err
	}

	// Now we validate the config
	err = validate(config)
	if err != nil {
		return nil, err
	}

	// Save generated config
	if l.generatedLoader != nil {
		err = l.generatedLoader.Save(generatedConfig)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

// SetDevSpaceRoot checks the current directory and all parent directories for a .devspace folder with a config and sets the current working directory accordingly
func (l *configLoader) SetDevSpaceRoot() (bool, error) {
	if l.options.ConfigPath != "" {
		configExists := configExistsInPath(l.options.ConfigPath)
		if !configExists {
			return configExists, nil
		}

		err := os.Chdir(filepath.Dir(l.ConfigPath()))
		if err != nil {
			return false, err
		}
		l.options.ConfigPath = filepath.Base(l.ConfigPath())
		return true, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return false, err
	}

	originalCwd := cwd
	homedir, err := homedir.Dir()
	if err != nil {
		return false, err
	}

	lastLength := 0
	for len(cwd) != lastLength {
		if cwd != homedir {
			configExists := configExistsInPath(filepath.Join(cwd, constants.DefaultConfigPath))
			if configExists {
				// Change working directory
				err = os.Chdir(cwd)
				if err != nil {
					return false, err
				}

				// Notify user that we are not using the current working directory
				if originalCwd != cwd {
					l.log.Infof("Using devspace config in %s", filepath.ToSlash(cwd))
				}

				return true, nil
			}
		}

		lastLength = len(cwd)
		cwd = filepath.Dir(cwd)
	}

	return false, nil
}
