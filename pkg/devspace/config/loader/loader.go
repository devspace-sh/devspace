package loader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"github.com/loft-sh/devspace/pkg/util/encoding"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"mvdan.cc/sh/v3/expand"

	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/remotecache"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/utils/pkg/command"

	"github.com/loft-sh/devspace/pkg/util/constraint"

	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/expression"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/vars"
)

// DefaultCommandVersionRegEx is the default regex to use if no regex is specified for determining the commands version
var DefaultCommandVersionRegEx = "(v\\d+\\.\\d+\\.\\d+)"

// ConfigLoader is the base interface for the main config loader
type ConfigLoader interface {
	// Load loads the devspace.yaml, parses it, applies profiles, fills in variables and
	// finally returns it.
	Load(ctx context.Context, client kubectl.Client, options *ConfigOptions, log log.Logger) (config.Config, error)

	// LoadWithCache loads the devspace.yaml, parses it, applies profiles, fills in variables and
	// finally returns it.
	LoadWithCache(ctx context.Context, localCache localcache.Cache, client kubectl.Client, options *ConfigOptions, log log.Logger) (config.Config, error)

	// LoadWithParser loads the config with the given parser
	LoadWithParser(ctx context.Context, localCache localcache.Cache, client kubectl.Client, parser Parser, options *ConfigOptions, log log.Logger) (config.Config, error)

	// LoadRaw loads the config without parsing it.
	LoadRaw() (map[string]interface{}, error)

	// LoadLocalCache loads the local cache from this config loader
	LoadLocalCache() (localcache.Cache, error)

	// Exists returns if a devspace.yaml could be found
	Exists() bool

	// ConfigPath returns the absolute devspace.yaml path for this config loader
	ConfigPath() string

	// SetDevSpaceRoot searches for a devspace.yaml in the current directory and parent directories
	// and will return if a devspace.yaml was found as well as switch to the current
	// working directory to that directory if a devspace.yaml could be found.
	SetDevSpaceRoot(log log.Logger) (bool, error)
}

type configLoader struct {
	absConfigPath string
}

// NewConfigLoader creates a new config loader with the given options
func NewConfigLoader(configPath string) (ConfigLoader, error) {
	if configPath == "" {
		configPath = os.Getenv("DEVSPACE_CONFIG")
	}

	absPath, err := filepath.Abs(ConfigPath(configPath))
	if err != nil {
		return nil, err
	}

	return &configLoader{
		absConfigPath: absPath,
	}, nil
}

func (l *configLoader) ConfigPath() string {
	return l.absConfigPath
}

func (l *configLoader) LoadLocalCache() (localcache.Cache, error) {
	return localcache.NewCacheLoader().Load(l.absConfigPath)
}

// Load restores variables from the cluster (if wanted), loads the config and then saves them to the cluster again
func (l *configLoader) Load(ctx context.Context, client kubectl.Client, options *ConfigOptions, log log.Logger) (config.Config, error) {
	return l.LoadWithCache(ctx, nil, client, options, log)
}

// LoadWithCache loads the config with the given local cache
func (l *configLoader) LoadWithCache(ctx context.Context, localCache localcache.Cache, client kubectl.Client, options *ConfigOptions, log log.Logger) (config.Config, error) {
	return l.LoadWithParser(ctx, localCache, client, NewDefaultParser(), options, log)
}

// LoadWithParser loads the config with the given parser
func (l *configLoader) LoadWithParser(ctx context.Context, localCache localcache.Cache, client kubectl.Client, parser Parser, options *ConfigOptions, log log.Logger) (_ config.Config, err error) {
	if localCache == nil {
		localCache, err = l.LoadLocalCache()
		if err != nil {
			return nil, err
		}
	}
	if options == nil {
		options = &ConfigOptions{}
	}

	defer func() {
		if err != nil {
			pluginErr := plugin.ExecutePluginHookWithContext(map[string]interface{}{"ERROR": err, "LOAD_PATH": l.absConfigPath}, "config.errorLoad", "error:loadConfig")
			if pluginErr != nil {
				log.Warnf("Error executing plugin hook: %v", pluginErr)
				return
			}
		}
	}()

	// call plugin hook
	pluginErr := plugin.ExecutePluginHookWithContext(map[string]interface{}{"LOAD_PATH": l.absConfigPath}, "config.beforeLoad", "before:loadConfig")
	if pluginErr != nil {
		return nil, pluginErr
	}

	// load the raw data
	data, err := l.LoadRaw()
	if err != nil {
		return nil, err
	}

	// make sure name is in config
	name := options.OverrideName
	if name == "" {
		var ok bool
		name, ok = data["name"].(string)
		if !ok {
			return nil, fmt.Errorf("name is missing in " + filepath.Base(l.absConfigPath))
		}
	} else {
		data["name"] = options.OverrideName
	}

	// validate name
	if encoding.IsUnsafeName(name) {
		return nil, fmt.Errorf("DevSpace config has an invalid name '%s', must match regex %s", name, encoding.UnsafeNameRegEx.String())
	}

	// set name to context
	ctx = values.WithName(ctx, name)

	// create remote cache
	var remoteCache remotecache.Cache
	if client != nil {
		remoteCache, err = remotecache.NewCacheLoader(name).Load(ctx, client)
		if err != nil {
			return nil, fmt.Errorf("error trying to load remote cache from current context and namespace: %v", err)
		}
	}

	parsedConfig, rawBeforeConversion, resolver, err := l.parseConfig(ctx, data, localCache, remoteCache, client, parser, options, log)
	if err != nil {
		return nil, err
	}

	err = l.ensureRequires(ctx, parsedConfig, log)
	if err != nil {
		return nil, errors.Wrap(err, "require versions")
	}

	c := config.NewConfig(data, rawBeforeConversion, parsedConfig, localCache, remoteCache, resolver.ResolvedVariables(), l.absConfigPath)
	pluginErr = plugin.ExecutePluginHookWithContext(map[string]interface{}{
		"LOAD_PATH":     l.absConfigPath,
		"LOADED_CONFIG": c.Config(),
		"LOADED_VARS":   c.Variables(),
		"LOADED_RAW":    c.Raw(),
	}, "config.afterLoad", "after:loadConfig")
	if pluginErr != nil {
		return nil, pluginErr
	}
	plugin.SetPluginConfig(c)
	return c, nil
}

func (l *configLoader) ensureRequires(ctx context.Context, config *latest.Config, log log.Logger) error {
	if config == nil {
		return nil
	}

	var aggregatedErrors []error

	if config.Require.DevSpace != "" {
		parsedConstraint, err := constraint.NewConstraint(config.Require.DevSpace)
		if err != nil {
			return errors.Wrap(err, "parsing require.devspace")
		}

		v, err := constraint.NewSemver(upgrade.GetVersion())
		if err != nil {
			return errors.Wrap(err, "parsing devspace version")
		}

		if !parsedConstraint.Check(v) {
			aggregatedErrors = append(aggregatedErrors, fmt.Errorf("DevSpace version mismatch: %s (currently installed) does not match %s (required by config). Please make sure you have installed DevSpace with version %s", upgrade.GetVersion(), config.Require.DevSpace, config.Require.DevSpace))
		}
	}

	if len(config.Require.Plugins) > 0 {
		pluginClient := plugin.NewClient(log)
		for index, p := range config.Require.Plugins {
			_, metadata, err := pluginClient.GetByName(p.Name)
			if err != nil {
				aggregatedErrors = append(aggregatedErrors, fmt.Errorf("cannot find plugin '%s' (%v), however it is required by the config. Please make sure you have installed the plugin '%s' with version %s", p.Name, err, p.Name, p.Version))
				continue
			} else if metadata == nil {
				aggregatedErrors = append(aggregatedErrors, fmt.Errorf("cannot find plugin '%s', however it is required by the config. Please make sure you have installed the plugin '%s' with version %s", p.Name, p.Name, p.Version))
				continue
			}

			parsedConstraint, err := constraint.NewConstraint(p.Version)
			if err != nil {
				return errors.Wrapf(err, "parsing require.plugins[%d].version", index)
			}

			v, err := constraint.NewSemver(metadata.Version)
			if err != nil {
				return errors.Wrapf(err, "parsing plugin %s version", p.Name)
			}

			if !parsedConstraint.Check(v) {
				aggregatedErrors = append(aggregatedErrors, fmt.Errorf("plugin '%s' version mismatch: %s (currently installed) does not match %s (required by config). Please make sure you have installed the plugin '%s' with version %s", p.Name, metadata.Version, p.Version, p.Name, p.Version))
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

		parsedConstraint, err := constraint.NewConstraint(c.Version)
		if err != nil {
			return errors.Wrapf(err, "parsing require.commands[%d].version", index)
		}

		out, err := command.Output(ctx, filepath.Dir(l.absConfigPath), expand.ListEnviron(os.Environ()...), c.Name, versionArgs...)
		if err != nil {
			aggregatedErrors = append(aggregatedErrors, fmt.Errorf("cannot run command '%s' (%v), however it is required by the config. Please make sure you have correctly installed '%s' with version %s", c.Name, err, c.Name, c.Version))
			continue
		}

		matches := regEx.FindStringSubmatch(string(out))
		if len(matches) != 2 {
			return fmt.Errorf("command %s %s output does not match the provided regex '%s', however the command is required by the config. Please make sure you have correctly installed '%s' with version %s", c.Name, strings.Join(versionArgs, " "), regExString, c.Name, c.Version)
		}

		v, err := constraint.NewSemver(matches[1])
		if err != nil {
			return fmt.Errorf("command %s %s output does not return a semver version, however the command is required by the config. Please make sure you have correctly installed '%s' with version %s", c.Name, strings.Join(versionArgs, " "), c.Name, c.Version)
		}

		if !parsedConstraint.Check(v) {
			aggregatedErrors = append(aggregatedErrors, fmt.Errorf("command '%s' version mismatch: %s (currently installed) does not match %s (required by config). Please make sure you have correctly installed '%s' with version %s", c.Name, matches[1], c.Version, c.Name, c.Version))
		}
	}

	return kerrors.NewAggregate(aggregatedErrors)
}

func (l *configLoader) parseConfig(
	ctx context.Context,
	rawConfig map[string]interface{},
	localCache localcache.Cache,
	remoteCache remotecache.Cache,
	client kubectl.Client,
	parser Parser,
	options *ConfigOptions,
	log log.Logger,
) (*latest.Config, map[string]interface{}, variable.Resolver, error) {
	// create a new variable resolver
	resolver, err := variable.NewResolver(localCache, &variable.PredefinedVariableOptions{
		ConfigPath: l.absConfigPath,
		KubeClient: client,
		Profile:    options.Profiles,
	}, options.Vars, log)
	if err != nil {
		return nil, nil, nil, err
	}

	// copy raw config
	copiedRawConfig, err := ResolveImports(ctx, resolver, filepath.Dir(l.absConfigPath), rawConfig, log)
	if err != nil {
		return nil, nil, nil, err
	}

	// Delete imports from config
	delete(copiedRawConfig, "imports")

	// prepare profiles
	copiedRawConfig, err = prepareProfiles(ctx, copiedRawConfig, resolver)
	if err != nil {
		return nil, nil, nil, err
	}

	// apply the profiles
	copiedRawConfig, err = l.applyProfiles(ctx, copiedRawConfig, options, resolver, log)
	if err != nil {
		return nil, nil, nil, err
	}

	// reload variables to make sure they are loaded correctly
	err = reloadVariables(resolver, copiedRawConfig, log)
	if err != nil {
		return nil, nil, nil, err
	}

	// Delete vars from config
	delete(copiedRawConfig, "vars")

	// parse the config
	latestConfig, rawBeforeConversion, err := parser.Parse(ctx, rawConfig, copiedRawConfig, resolver, log)
	if err != nil {
		return nil, nil, nil, err
	}

	// check if we do not want to change the generated config or
	// secret vars.
	if options.Dry {
		return latestConfig, rawBeforeConversion, resolver, nil
	}

	// save local cache
	err = localCache.Save()
	if err != nil {
		return nil, nil, nil, err
	}

	return latestConfig, rawBeforeConversion, resolver, nil
}

func reloadVariables(resolver variable.Resolver, rawConfig map[string]interface{}, log log.Logger) error {
	// Load defined variables again (might be changed through profiles)
	loadedVars, err := versions.ParseVariables(rawConfig, log)
	if err != nil {
		return err
	}

	// update the used vars in the resolver
	resolver.UpdateVars(loadedVars)
	return nil
}

func validateProfile(profile interface{}) error {
	profileMap, ok := profile.(map[string]interface{})
	if !ok {
		return fmt.Errorf("profile is not an object")
	}

	parents := profileMap["parents"]
	if parents != nil {
		parentsString, ok := parents.(string)
		if ok {
			if vars.VarMatchRegex.MatchString(parentsString) {
				return fmt.Errorf("parents cannot be a variable")
			}

			if expression.ExpressionMatchRegex.MatchString(parentsString) {
				return fmt.Errorf("parents cannot be an expression")
			}

			return fmt.Errorf("parents is not an array")
		}
	}

	activation := profileMap["activation"]
	if activation != nil {
		activationString, ok := activation.(string)
		if ok {
			if vars.VarMatchRegex.MatchString(activationString) {
				return fmt.Errorf("activation cannot be a variable")
			}

			if expression.ExpressionMatchRegex.MatchString(activationString) {
				return fmt.Errorf("activation cannot be an expression")
			}

			return fmt.Errorf("activation is not an array")
		}
	}

	profileConfig, err := copyForValidation(profile)
	if err != nil {
		return err
	}

	if vars.VarMatchRegex.MatchString(profileConfig.Name) {
		return fmt.Errorf("name cannot be a variable")
	}

	if expression.ExpressionMatchRegex.MatchString(profileConfig.Name) {
		return fmt.Errorf("name cannot be an expression")
	}

	if vars.VarMatchRegex.MatchString(profileConfig.Parent) {
		return fmt.Errorf("parent cannot be a variable")
	}

	if expression.ExpressionMatchRegex.MatchString(profileConfig.Parent) {
		return fmt.Errorf("parent cannot be an expression")
	}

	for idx, patch := range profileConfig.Patches {
		if expression.ExpressionMatchRegex.MatchString(patch.Path) {
			return fmt.Errorf("patches[%d] path cannot be an expression", idx)
		}

		if vars.VarMatchRegex.MatchString(patch.Operation) {
			return fmt.Errorf("patches[%d] op cannot be a variable", idx)
		}

		if expression.ExpressionMatchRegex.MatchString(patch.Operation) {
			return fmt.Errorf("patches[%d] op cannot be an expression", idx)
		}
	}

	return nil
}

func prepareProfiles(ctx context.Context, config map[string]interface{}, resolver variable.Resolver) (map[string]interface{}, error) {
	rawProfiles := config["profiles"]
	if rawProfiles == nil {
		return config, nil
	}

	resolved, err := resolve(ctx, rawProfiles, resolver)
	if err != nil {
		return nil, err
	}

	profiles, ok := resolved.([]interface{})
	if !ok {
		return nil, fmt.Errorf("error validating profiles: not an array")
	}

	for idx, profile := range profiles {
		resolvedProfile, err := resolve(ctx, profile, resolver)
		if err != nil {
			return nil, err
		}

		profileMap, ok := resolvedProfile.(map[string]interface{})
		if !ok {
			return nil, errors.Wrapf(err, "error resolving profiles[%d], object expected", idx)
		}

		// Resolve merge field
		if profileMap["merge"] != nil {
			merge, err := resolve(ctx, profileMap["merge"], resolver)
			if err != nil {
				return nil, err
			}
			profileMap["merge"] = merge
		}

		// Resolve patches field
		if profileMap["patches"] != nil {
			patches, err := resolve(ctx, profileMap["patches"], resolver)
			if err != nil {
				return nil, err
			}
			profileMap["patches"] = patches
		}

		// Resolve replace field
		if profileMap["replace"] != nil {
			replace, err := resolve(ctx, profileMap["replace"], resolver)
			if err != nil {
				return nil, err
			}
			profileMap["replace"] = replace
		}

		// Validate that the profile doesn't use forbidden expressions
		err = validateProfile(profileMap)
		if err != nil {
			return nil, errors.Wrapf(err, "error validating profiles[%d]", idx)
		}

		profiles[idx] = profileMap
	}

	config["profiles"] = profiles

	return config, nil
}

func resolve(ctx context.Context, data interface{}, resolver variable.Resolver) (interface{}, error) {
	_, ok := data.(string)
	if !ok {
		return data, nil
	}

	// find and fill variables
	return resolver.FillVariables(ctx, data, true)
}

func (l *configLoader) applyProfiles(ctx context.Context, data map[string]interface{}, options *ConfigOptions, resolver variable.Resolver, log log.Logger) (map[string]interface{}, error) {
	// Get profile
	profiles, err := versions.ParseProfile(ctx, filepath.Dir(l.absConfigPath), data, options.Profiles, options.ProfileRefresh, options.DisableProfileActivation, resolver, log)
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

		// Apply patches
		data, err = ApplyPatches(data, profiles[i])
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

// configExistsInPath checks whether a devspace configuration exists at a certain path
func configExistsInPath(path string) bool {
	_, err := os.Stat(path)
	return err == nil // false, no config file found
}

// LoadRaw loads the raw config
func (l *configLoader) LoadRaw() (map[string]interface{}, error) {
	// What path should we use
	configPath := ConfigPath(l.absConfigPath)
	_, err := os.Stat(configPath)
	if err != nil {
		return nil, errors.Errorf("Couldn't load '%s': %v", configPath, err)
	}

	fileContent, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	rawMap := map[string]interface{}{}
	err = yamlutil.Unmarshal(fileContent, &rawMap)
	if err != nil {
		return nil, err
	}

	name, ok := rawMap["name"].(string)
	if !ok || name == "" {
		directoryName := filepath.Base(filepath.Dir(l.absConfigPath))
		if directoryName != "" && len(directoryName) > 2 {
			name = encoding.Convert(directoryName)
		} else {
			name = "devspace"
		}

		rawMap["name"] = name
	}

	return rawMap, nil
}

// Exists checks whether the yaml file for the config exists or the configs.yaml exists
func (l *configLoader) Exists() bool {
	return configExistsInPath(ConfigPath(l.absConfigPath))
}

// SetDevSpaceRoot checks the current directory and all parent directories for a .devspace folder with a config and sets the current working directory accordingly
func (l *configLoader) SetDevSpaceRoot(log log.Logger) (bool, error) {
	if l.absConfigPath != "" {
		configExists := configExistsInPath(l.absConfigPath)
		if !configExists {
			return configExists, nil
		}

		err := os.Chdir(filepath.Dir(l.absConfigPath))
		if err != nil {
			return false, err
		}

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

func copyRaw(in map[string]interface{}) (map[string]interface{}, error) {
	o, err := yaml.Marshal(in)
	if err != nil {
		return nil, err
	}

	n := map[string]interface{}{}
	err = yamlutil.Unmarshal(o, &n)
	if err != nil {
		return nil, err
	}

	return n, nil
}

func copyForValidation(profile interface{}) (*latest.ProfileConfig, error) {
	profileMap, ok := profile.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error loading profiles: invalid format")
	}

	clone := map[string]interface{}{
		"name":       profileMap["name"],
		"parent":     profileMap["parent"],
		"parents":    profileMap["parents"],
		"patches":    profileMap["patches"],
		"activation": profileMap["activation"],
	}

	o, err := yaml.Marshal(clone)
	if err != nil {
		return nil, err
	}

	profileConfig := &latest.ProfileConfig{}
	err = yamlutil.UnmarshalStrict(o, profileConfig)
	if err != nil {
		return nil, err
	}

	return profileConfig, nil
}
