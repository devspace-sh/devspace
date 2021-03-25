package loader

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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

	// LoadWithParser loads the config with the given parser
	LoadWithParser(parser Parser, options *ConfigOptions, log log.Logger) (config.Config, error)

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
		configPath:       configPath,
		kubeConfigLoader: kubeconfig.NewLoader(),
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

// Load restores variables from the cluster (if wanted), loads the config and then saves them to the cluster again
func (l *configLoader) Load(options *ConfigOptions, log log.Logger) (config.Config, error) {
	return l.LoadWithParser(NewDefaultParser(), options, log)
}

// LoadWithParser loads the config with the given parser
func (l *configLoader) LoadWithParser(parser Parser, options *ConfigOptions, log log.Logger) (config.Config, error) {
	if options == nil {
		options = &ConfigOptions{}
	}

	data, err := l.LoadRaw()
	if err != nil {
		return nil, err
	}

	parsedConfig, generatedConfig, resolver, err := l.parseConfig(data, parser, options, log)
	if err != nil {
		return nil, err
	}

	return config.NewConfig(data, parsedConfig, generatedConfig, resolver.ResolvedVariables()), nil
}

func (l *configLoader) parseConfig(rawConfig map[interface{}]interface{}, parser Parser, options *ConfigOptions, log log.Logger) (*latest.Config, *generated.Config, variable.Resolver, error) {
	// load the generated config
	generatedConfig, err := l.LoadGenerated(options)
	if err != nil {
		return nil, nil, nil, err
	}

	// restore vars if wanted
	if options.KubeClient != nil && options.RestoreVars {
		vars, _, err := RestoreVarsFromSecret(options.KubeClient, options.VarsSecretName)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "restore vars")
		} else if vars != nil {
			generatedConfig.Vars = vars
		}
	}

	// check if we should load the profile from the generated config
	if generatedConfig.ActiveProfile != "" && options.Profile == "" {
		options.Profile = generatedConfig.ActiveProfile
	}

	// create a new variable resolver
	resolver := l.newVariableResolver(generatedConfig, options, log)

	// copy raw config
	copiedRawConfig, err := config.CopyRaw(rawConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	// apply the profiles
	copiedRawConfig, err = l.applyProfiles(copiedRawConfig, options, log)
	if err != nil {
		return nil, nil, nil, err
	}

	// Load defined variables
	vars, err := versions.ParseVariables(copiedRawConfig, log)
	if err != nil {
		return nil, nil, nil, err
	}

	// Delete vars from config
	delete(copiedRawConfig, "vars")

	// parse the config
	latestConfig, err := parser.Parse(copiedRawConfig, vars, resolver, options, log)
	if err != nil {
		return nil, nil, nil, err
	}

	// now we validate the config
	err = validate(latestConfig, log)
	if err != nil {
		return nil, nil, nil, err
	}

	// Save generated config
	if options.generatedLoader == nil {
		err = generated.NewConfigLoader(options.Profile).Save(generatedConfig)
	} else {
		err = options.generatedLoader.Save(generatedConfig)
	}
	if err != nil {
		return nil, nil, nil, err
	}

	// save vars if wanted
	if options.KubeClient != nil && options.SaveVars {
		err = SaveVarsInSecret(options.KubeClient, generatedConfig.Vars, options.VarsSecretName, log)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "save vars")
		}
	}

	return latestConfig, generatedConfig, resolver, nil
}

func (l *configLoader) applyProfiles(data map[interface{}]interface{}, options *ConfigOptions, log log.Logger) (map[interface{}]interface{}, error) {
	// Get profile
	profiles, err := versions.ParseProfile(filepath.Dir(l.configPath), data, options.Profile, options.ProfileParents, options.ProfileRefresh, log)
	if err != nil {
		return nil, err
	}

	// Now delete not needed parts from config
	delete(data, "profiles")

	// Apply profiles
	for i := len(profiles) - 1; i >= 0; i-- {
		// Apply replace
		err = ApplyReplace(data, profiles[i])
		if err != nil {
			return nil, err
		}

		// Apply merge
		data, err = ApplyMerge(data, profiles[i])
		if err != nil {
			return nil, err
		}

		// Apply strategic merge
		data, err = ApplyStrategicMerge(data, profiles[i])
		if err != nil {
			return nil, err
		}

		// Apply patches
		data, err = ApplyPatches(data, profiles[i])
		if err != nil {
			return nil, err
		}
	}

	return data, nil
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

// fillVariables fills in the given vars into the prepared config
func fillVariables(resolver variable.Resolver, preparedConfig map[interface{}]interface{}, vars []*latest.Variable, options *ConfigOptions) error {
	// Find out what vars are really used
	varsUsed, err := resolver.FindVariables(preparedConfig, vars)
	if err != nil {
		return err
	}

	// parse cli --var's, the resolver will cache them for us
	_, err = resolver.ConvertFlags(options.Vars)
	if err != nil {
		return err
	}

	// Fill used defined variables
	if len(vars) > 0 {
		newVars := []*latest.Variable{}
		for _, v := range vars {
			if varsUsed[strings.TrimSpace(v.Name)] {
				newVars = append(newVars, v)
			}
		}

		if len(newVars) > 0 {
			err = askQuestions(resolver, newVars)
			if err != nil {
				return err
			}
		}
	}

	// Walk over data and fill in variables
	err = resolver.FillVariables(preparedConfig)
	if err != nil {
		return err
	}

	return nil
}

func askQuestions(resolver variable.Resolver, vars []*latest.Variable) error {
	for _, definition := range vars {
		name := strings.TrimSpace(definition.Name)

		// fill the variable with definition
		_, err := resolver.Resolve(name, definition)
		if err != nil {
			return err
		}
	}

	return nil
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

func removeCommands(data map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	delete(data, "commands")

	return data, nil
}
