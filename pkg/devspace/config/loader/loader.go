package loader

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// ConfigLoader is the base interface for the main config loader
type ConfigLoader interface {
	// Load loads the devspace.yaml, parses it, applies profiles, fills in variables and
	// finally returns it.
	Load(options *ConfigOptions, log log.Logger) (config.Config, error)

	// LoadRaw loads the config without parsing it.
	LoadRaw() (map[interface{}]interface{}, error)

	// LoadCommands loads only the devspace commands from the devspace.yaml
	LoadCommands(options *ConfigOptions, log log.Logger) ([]*latest.CommandConfig, error)

	// LoadGenerated loads the generated config
	LoadGenerated(options *ConfigOptions) (*generated.Config, error)

	// Saves a devspace.yaml to file
	Save(config *latest.Config) error

	// Saves a generated config yaml to file
	SaveGenerated(generated *generated.Config) error

	// Returns if a devspace.yaml could be found
	Exists() bool

	// Searches for a devspace.yaml in the current directory and parent directories
	// and will return if a devspace.yaml was found as well as switch to the current
	// working directory to that directory if a devspace.yaml could be found.
	SetDevSpaceRoot(log log.Logger) (bool, error)
}

type configLoader struct {
	kubeConfigLoader kubeconfig.Loader

	configPath string
}

// NewConfigLoader creates a new config loader with the given options
func NewConfigLoader(configPath string) ConfigLoader {
	return &configLoader{
		configPath: configPath,
	}
}

// LoadGenerated loads the generated config from file
func (l *configLoader) LoadGenerated(options *ConfigOptions) (*generated.Config, error) {
	var err error
	if options == nil {
		options = &ConfigOptions{}
	}

	generatedConfig := options.GeneratedConfig
	if generatedConfig == nil {
		if options.generatedLoader == nil {
			generatedConfig, err = generated.NewConfigLoader(options.Profile).Load()
		} else {
			generatedConfig, err = options.generatedLoader.Load()
		}
		if err != nil {
			return nil, err
		}
	}

	return generatedConfig, nil
}

// RestoreLoadSave restores variables from the cluster (if wanted), loads the config and then saves them to the cluster again
func (l *configLoader) Load(options *ConfigOptions, log log.Logger) (config.Config, error) {
	if options == nil {
		options = &ConfigOptions{}
	}

	// load the generated config
	generatedConfig, err := l.LoadGenerated(options)
	if err != nil {
		return nil, err
	}

	// restore vars if wanted
	if options.KubeClient != nil && options.RestoreVars {
		vars, _, err := RestoreVarsFromSecret(options.KubeClient, options.VarsSecretName)
		if err != nil {
			return nil, errors.Wrap(err, "restore vars")
		} else if vars != nil {
			generatedConfig.Vars = vars
		}
	}

	// check if we should load the profile from the generated config
	if generatedConfig.ActiveProfile != "" && options.Profile == "" {
		options.Profile = generatedConfig.ActiveProfile
	}

	// load the raw config
	rawConfig, err := l.LoadRaw()
	if err != nil {
		return nil, err
	}

	// copy raw config
	copiedRawConfig, err := config.CopyRaw(rawConfig)
	if err != nil {
		return nil, err
	}

	// create a new variable resolver
	resolver := l.newVariableResolver(generatedConfig, options, log)

	// parse the config
	parsedConfig, err := l.parseConfig(resolver, rawConfig, options, log)
	if err != nil {
		return nil, err
	}

	// now we validate the config
	err = validate(parsedConfig, log)
	if err != nil {
		return nil, err
	}

	// Save generated config
	if options.generatedLoader == nil {
		err = generated.NewConfigLoader(options.Profile).Save(generatedConfig)
	} else {
		err = options.generatedLoader.Save(generatedConfig)
	}
	if err != nil {
		return nil, err
	}

	// save vars if wanted
	if options.KubeClient != nil && options.SaveVars {
		err = SaveVarsInSecret(options.KubeClient, generatedConfig.Vars, options.VarsSecretName, log)
		if err != nil {
			return nil, errors.Wrap(err, "save vars")
		}
	}

	return config.NewConfig(copiedRawConfig, parsedConfig, generatedConfig, resolver.ResolvedVariables()), nil
}

// LoadCommands fills the variables in the data and parses the commands
func (l *configLoader) LoadCommands(options *ConfigOptions, log log.Logger) ([]*latest.CommandConfig, error) {
	data, err := l.LoadRaw()
	if err != nil {
		return nil, err
	}

	// apply the profiles
	data, err = l.applyProfiles(data, options, log)
	if err != nil {
		return nil, err
	}

	// Load defined variables
	vars, err := versions.ParseVariables(data, log)
	if err != nil {
		return nil, err
	}

	// Parse commands
	preparedConfig, err := versions.ParseCommands(data)
	if err != nil {
		return nil, err
	}

	// load the generated config
	generatedConfig, err := l.LoadGenerated(options)
	if err != nil {
		return nil, err
	}

	// Fill in variables
	err = l.fillVariables(l.newVariableResolver(generatedConfig, options, log), preparedConfig, vars, options)
	if err != nil {
		return nil, err
	}

	// Now parse the whole config
	parsedConfig, err := versions.Parse(preparedConfig, log)
	if err != nil {
		return nil, errors.Wrap(err, "parse config")
	}

	return parsedConfig.Commands, nil
}

func (l *configLoader) newVariableResolver(generatedConfig *generated.Config, options *ConfigOptions, log log.Logger) variable.Resolver {
	return variable.NewResolver(generatedConfig.Vars, &variable.PredefinedVariableOptions{
		BasePath:         options.BasePath,
		ConfigPath:       ConfigPath(l.configPath),
		KubeContextFlag:  options.KubeContext,
		NamespaceFlag:    options.Namespace,
		KubeConfigLoader: l.kubeConfigLoader,
		Profile:          options.Profile,
	}, log)
}

// SaveGenerated is a convenience method to save the generated config
func (l *configLoader) SaveGenerated(generatedConfig *generated.Config) error {
	return generated.NewConfigLoader("").Save(generatedConfig)
}

// configExistsInPath checks whether a devspace configuration exists at a certain path
func configExistsInPath(path string) bool {
	// check devspace.yaml
	_, err := os.Stat(path)
	if err == nil {
		return true
	}

	return false // no config file found
}

// LoadRaw loads the raw config
func (l *configLoader) LoadRaw() (map[interface{}]interface{}, error) {
	// What path should we use
	configPath := ConfigPath(l.configPath)
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

// Exists checks whether the yaml file for the config exists or the configs.yaml exists
func (l *configLoader) Exists() bool {
	return configExistsInPath(ConfigPath(l.configPath))
}

// SetDevSpaceRoot checks the current directory and all parent directories for a .devspace folder with a config and sets the current working directory accordingly
func (l *configLoader) SetDevSpaceRoot(log log.Logger) (bool, error) {
	if l.configPath != "" {
		configExists := configExistsInPath(l.configPath)
		if !configExists {
			return configExists, nil
		}

		err := os.Chdir(filepath.Dir(ConfigPath(l.configPath)))
		if err != nil {
			return false, err
		}
		l.configPath = filepath.Base(ConfigPath(l.configPath))
		return true, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return false, err
	}

	originalCwd := cwd
	homeDir, err := homedir.Dir()
	if err != nil {
		return false, err
	}

	lastLength := 0
	for len(cwd) != lastLength {
		if cwd != homeDir {
			configExists := configExistsInPath(filepath.Join(cwd, constants.DefaultConfigPath))
			if configExists {
				// Change working directory
				err = os.Chdir(cwd)
				if err != nil {
					return false, err
				}

				// Notify user that we are not using the current working directory
				if originalCwd != cwd {
					log.Infof("Using devspace config in %s", filepath.ToSlash(cwd))
				}

				return true, nil
			}
		}

		lastLength = len(cwd)
		cwd = filepath.Dir(cwd)
	}

	return false, nil
}

func ConfigPath(configPath string) string {
	path := constants.DefaultConfigPath
	if configPath != "" {
		path = configPath
	}

	return path
}
