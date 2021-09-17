package loader

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
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

	version "github.com/hashicorp/go-version"
)

// DefaultCommandVersionRegEx is the default regex to use if no regex is specified for determining the commands version
var DefaultCommandVersionRegEx = "(v\\d+\\.\\d+\\.\\d+)"

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

	// Save saves a devspace.yaml to file
	Save(config *latest.Config) error

	// SaveGenerated saves a generated config yaml to file
	SaveGenerated(generated *generated.Config) error

	// Exists returns if a devspace.yaml could be found
	Exists() bool

	// SetDevSpaceRoot searches for a devspace.yaml in the current directory and parent directories
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
		if options.GeneratedLoader == nil {
			generatedConfig, err = generated.NewConfigLoaderFromDevSpacePath(GetLastProfile(options.Profiles), l.configPath).Load()
		} else {
			generatedConfig, err = options.GeneratedLoader.Load()
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

	absPath, err := filepath.Abs(ConfigPath(l.configPath))
	if err != nil {
		return nil, err
	}

	// call plugin hook
	pluginErr := plugin.ExecutePluginHookWithContext("config.beforeLoad", map[string]interface{}{"LOAD_PATH": absPath})
	if pluginErr != nil {
		return nil, pluginErr
	}

	data, err := l.LoadRaw()
	if err != nil {
		pluginErr = plugin.ExecutePluginHookWithContext("config.errorLoad", map[string]interface{}{"ERROR": err, "LOAD_PATH": absPath})
		if pluginErr != nil {
			return nil, pluginErr
		}

		return nil, err
	}

	parsedConfig, generatedConfig, resolver, err := l.parseConfig(absPath, data, parser, options, log)
	if err != nil {
		pluginErr = plugin.ExecutePluginHookWithContext("config.errorLoad", map[string]interface{}{"ERROR": err, "LOAD_PATH": absPath})
		if pluginErr != nil {
			return nil, pluginErr
		}

		return nil, err
	}

	err = l.ensureRequires(parsedConfig, log)
	if err != nil {
		pluginErr = plugin.ExecutePluginHookWithContext("config.errorLoad", map[string]interface{}{"ERROR": err, "LOAD_PATH": absPath})
		if pluginErr != nil {
			return nil, pluginErr
		}

		return nil, errors.Wrap(err, "require versions")
	}

	c := config.NewConfig(data, parsedConfig, generatedConfig, resolver.ResolvedVariables(), absPath)
	pluginErr = plugin.ExecutePluginHookWithContext("config.afterLoad", map[string]interface{}{
		"LOAD_PATH":     absPath,
		"LOADED_CONFIG": c.Config(),
		"LOADED_VARS":   c.Variables(),
		"LOADED_RAW":    c.Raw(),
	})
	if pluginErr != nil {
		return nil, pluginErr
	}
	plugin.SetPluginConfig(c)
	return c, nil
}

func (l *configLoader) ensureRequires(config *latest.Config, log log.Logger) error {
	if config == nil {
		return nil
	}

	if config.Require.DevSpace != "" {
		constraint, err := version.NewConstraint(config.Require.DevSpace)
		if err != nil {
			return errors.Wrap(err, "parsing require.devspace")
		}

		v, err := version.NewSemver(upgrade.GetVersion())
		if err != nil {
			return errors.Wrap(err, "parsing devspace version")
		}

		if !constraint.Check(v) {
			return fmt.Errorf("DevSpace version mismatch: %s (currently installed) does not match %s (required by config). Please make sure you have installed DevSpace with version %s", upgrade.GetVersion(), config.Require.DevSpace, config.Require.DevSpace)
		}
	}

	if len(config.Require.Plugins) > 0 {
		pluginClient := plugin.NewClient(log)
		for index, p := range config.Require.Plugins {
			_, metadata, err := pluginClient.GetByName(p.Name)
			if err != nil {
				return fmt.Errorf("cannot find plugin '%s' (%v), however it is required by the config. Please make sure you have installed the plugin '%s' with version %s", p.Name, err, p.Name, p.Version)
			} else if metadata == nil {
				return fmt.Errorf("cannot find plugin '%s', however it is required by the config. Please make sure you have installed the plugin '%s' with version %s", p.Name, p.Name, p.Version)
			}

			constraint, err := version.NewConstraint(p.Version)
			if err != nil {
				return errors.Wrapf(err, "parsing require.plugins[%d].version", index)
			}

			v, err := version.NewSemver(metadata.Version)
			if err != nil {
				return errors.Wrapf(err, "parsing plugin %s version", p.Name)
			}

			if !constraint.Check(v) {
				return fmt.Errorf("plugin '%s' version mismatch: %s (currently installed) does not match %s (required by config). Please make sure you have installed the plugin '%s' with version %s", p.Name, metadata.Version, p.Version, p.Name, p.Version)
			}
		}
	}

	for index, c := range config.Require.Commands {
		regExString := c.VersionRegEx
		if regExString == "" {
			regExString = DefaultCommandVersionRegEx
		}

		versionArgs := c.VersionArgs
		if c.VersionArgs == nil {
			versionArgs = []string{"version"}
		}

		regEx, err := regexp.Compile(regExString)
		if err != nil {
			return errors.Wrapf(err, "parsing require.commands[%d].versionRegEx", index)
		}

		constraint, err := version.NewConstraint(c.Version)
		if err != nil {
			return errors.Wrapf(err, "parsing require.commands[%d].version", index)
		}

		out, err := exec.Command(c.Name, versionArgs...).Output()
		if err != nil {
			return fmt.Errorf("cannot run command '%s' (%v), however it is required by the config. Please make sure you have correctly installed '%s' with version %s", c.Name, err, c.Name, c.Version)
		}

		matches := regEx.FindStringSubmatch(string(out))
		if len(matches) != 2 {
			return fmt.Errorf("command %s %s output does not match the provided regex '%s', however the command is required by the config. Please make sure you have correctly installed '%s' with version %s", c.Name, strings.Join(versionArgs, " "), regExString, c.Name, c.Version)
		}

		v, err := version.NewSemver(matches[1])
		if err != nil {
			return fmt.Errorf("command %s %s output does not return a semver version, however the command is required by the config. Please make sure you have correctly installed '%s' with version %s", c.Name, strings.Join(versionArgs, " "), c.Name, c.Version)
		}

		if !constraint.Check(v) {
			return fmt.Errorf("command '%s' version mismatch: %s (currently installed) does not match %s (required by config). Please make sure you have correctly installed '%s' with version %s", c.Name, matches[1], c.Version, c.Name, c.Version)
		}
	}

	return nil
}

func (l *configLoader) parseConfig(absPath string, rawConfig map[interface{}]interface{}, parser Parser, options *ConfigOptions, log log.Logger) (*latest.Config, *generated.Config, variable.Resolver, error) {
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
	if generatedConfig.ActiveProfile != "" && len(options.Profiles) == 0 {
		options.Profiles = []string{generatedConfig.ActiveProfile}
	}

	// create a new variable resolver
	resolver := l.newVariableResolver(generatedConfig, options, log)

	// copy raw config
	copiedRawConfig, err := copyRaw(rawConfig)
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

	// validate variables
	err = validateVars(vars)
	if err != nil {
		return nil, nil, nil, err
	}

	// Delete vars from config
	delete(copiedRawConfig, "vars")

	// parse the config
	latestConfig, err := parser.Parse(absPath, rawConfig, copiedRawConfig, vars, resolver, options, log)
	if err != nil {
		return nil, nil, nil, err
	}

	// now we validate the config
	err = validate(latestConfig, log)
	if err != nil {
		return nil, nil, nil, err
	}

	// Save generated config
	if options.GeneratedLoader == nil {
		err = generated.NewConfigLoaderFromDevSpacePath(GetLastProfile(options.Profiles), l.configPath).Save(generatedConfig)
	} else {
		err = options.GeneratedLoader.Save(generatedConfig)
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
	profiles, err := versions.ParseProfile(filepath.Dir(l.configPath), data, options.Profiles, options.ProfileRefresh, options.DisableProfileActivation, log)
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
		Profile:          GetLastProfile(options.Profiles),
	}, log)
}

func GetLastProfile(profiles []string) string {
	if len(profiles) == 0 {
		return ""
	}
	return profiles[len(profiles)-1]
}

// configExistsInPath checks whether a devspace configuration exists at a certain path
func configExistsInPath(path string) bool {
	// check devspace.yaml
	_, err := os.Stat(path)
	return err == nil // false, no config file found
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

func copyRaw(in map[interface{}]interface{}) (map[interface{}]interface{}, error) {
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
