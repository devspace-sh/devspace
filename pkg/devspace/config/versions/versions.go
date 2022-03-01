package versions

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta11"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta10"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta1"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta2"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta3"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta4"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta5"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta6"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta7"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta8"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta9"
	dependencyutil "github.com/loft-sh/devspace/pkg/devspace/dependency/util"
	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"
)

type loader struct {
	New config.New
}

var versionLoader = map[string]*loader{
	v1beta1.Version:  {New: v1beta1.New},
	v1beta2.Version:  {New: v1beta2.New},
	v1beta3.Version:  {New: v1beta3.New},
	v1beta4.Version:  {New: v1beta4.New},
	v1beta5.Version:  {New: v1beta5.New},
	v1beta6.Version:  {New: v1beta6.New},
	v1beta7.Version:  {New: v1beta7.New},
	v1beta8.Version:  {New: v1beta8.New},
	v1beta9.Version:  {New: v1beta9.New},
	v1beta10.Version: {New: v1beta10.New},
	v1beta11.Version: {New: v1beta11.New},
	latest.Version:   {New: latest.New},
}

// ParseProfile loads the base config & a certain profile
func ParseProfile(ctx context.Context, basePath string, data map[string]interface{}, profiles []string, update bool, disableProfileActivation bool, resolver variable.Resolver, log log.Logger) ([]map[string]interface{}, error) {
	parsedProfiles := []map[string]interface{}{}

	// auto activated root level profiles
	activatedProfiles := []string{}
	if !disableProfileActivation {
		var err error
		activatedProfiles, err = getActivatedProfiles(data, resolver)
		if err != nil {
			return nil, err
		}
	}

	// Combine auto activated profiles with flag activated profiles
	profiles = append(activatedProfiles, profiles...)
	profiles = filterProfileParents(profiles)

	// check if there are profile parents
	for i := len(profiles) - 1; i >= 0; i-- {
		err := getProfiles(ctx, basePath, data, profiles[i], &parsedProfiles, 1, update, log)
		if err != nil {
			return nil, err
		}
	}

	return parsedProfiles, nil
}

// ParseImports parses only the commands from the config
func ParseImports(data map[string]interface{}) (map[string]interface{}, error) {
	retMap := map[string]interface{}{}
	err := util.Convert(data, &retMap)
	if err != nil {
		return nil, err
	}

	imports, ok := retMap["imports"]
	if !ok {
		return map[string]interface{}{
			"version": retMap["version"],
			"name":    retMap["name"],
		}, nil
	}

	return map[string]interface{}{
		"version": retMap["version"],
		"name":    retMap["name"],
		"imports": imports,
	}, nil
}

// ParseCommands parses only the commands from the config
func ParseCommands(data map[string]interface{}) (map[string]interface{}, error) {
	retMap := map[string]interface{}{}
	err := util.Convert(data, &retMap)
	if err != nil {
		return nil, err
	}

	commands, ok := retMap["commands"]
	if !ok {
		return map[string]interface{}{
			"version": retMap["version"],
			"name":    retMap["name"],
		}, nil
	}

	return map[string]interface{}{
		"version":  retMap["version"],
		"name":     retMap["name"],
		"commands": commands,
	}, nil
}

// ParseVariables parses only the variables from the config
func ParseVariables(data map[string]interface{}, log log.Logger) ([]*latest.Variable, error) {
	strippedData, err := getVariables(data)
	if err != nil {
		return nil, errors.Wrap(err, "loading variables")
	}

	config, err := Parse(strippedData, log)
	if err != nil {
		return nil, errors.Wrap(err, "parse variables")
	}

	return config.Vars, nil
}

// Parse parses the data into the latest config
func Parse(data map[string]interface{}, log log.Logger) (*latest.Config, error) {
	version, ok := data["version"].(string)
	if !ok {
		return nil, errors.Errorf("Version is missing in devspace.yaml")
	}

	loader, ok := versionLoader[version]
	if !ok {
		return nil, errors.Errorf("Unrecognized config version %s. Please upgrade devspace with `devspace upgrade`", version)
	}

	versionLoadFunc := loader.New

	// Load config strict
	latestConfig := versionLoadFunc()
	out, err := yaml.Marshal(data)
	if err != nil {
		return nil, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(out))
	decoder.KnownFields(true)
	err = decoder.Decode(latestConfig)
	if err != nil {
		return nil, errors.Errorf("Error loading config: %v", err)
	}

	// Upgrade config to latest
	for latestConfig.GetVersion() != latest.Version {
		upgradedConfig, err := latestConfig.Upgrade(log)
		if err != nil {
			return nil, errors.Errorf("Error upgrading config from version %s: %v", latestConfig.GetVersion(), err)
		}

		latestConfig = upgradedConfig
	}

	// Convert
	latestConfigConverted, ok := latestConfig.(*latest.Config)
	if !ok {
		return nil, errors.Errorf("Error converting config, latest config is not the latest version")
	}

	// Update version to latest
	latestConfigConverted.Version = latest.Version

	// Filter out empty images, deployments etc.
	adjustConfig(latestConfigConverted)

	return latestConfigConverted, nil
}

func adjustConfig(config *latest.Config) {
	for name, devPod := range config.Dev {
		devPod.Name = name
	}
	for name, pipeline := range config.Pipelines {
		pipeline.Name = name
	}
	if config.Images != nil {
		newObjs := map[string]*latest.Image{}
		for k, v := range config.Images {
			if v != nil {
				newObjs[k] = v
			}
		}
		config.Images = newObjs
	}
	if config.Deployments != nil {
		newObjs := map[string]*latest.DeploymentConfig{}
		for k, v := range config.Deployments {
			if v != nil {
				v.Name = k
				newObjs[k] = v
			}
		}
		config.Deployments = newObjs
	}
	if config.Dependencies != nil {
		newObjs := []*latest.DependencyConfig{}
		for _, v := range config.Dependencies {
			if v != nil {
				newObjs = append(newObjs, v)
			}
		}
		config.Dependencies = newObjs
	}
	if config.Hooks != nil {
		newObjs := []*latest.HookConfig{}
		for _, v := range config.Hooks {
			if v != nil {
				newObjs = append(newObjs, v)
			}
		}
		config.Hooks = newObjs
	}
	if config.PullSecrets != nil {
		newObjs := []*latest.PullSecretConfig{}
		for _, v := range config.PullSecrets {
			if v != nil {
				newObjs = append(newObjs, v)
			}
		}
		config.PullSecrets = newObjs
	}
	if config.Commands != nil {
		newObjs := []*latest.CommandConfig{}
		for _, v := range config.Commands {
			if v != nil {
				newObjs = append(newObjs, v)
			}
		}
		config.Commands = newObjs
	}
	if config.Open != nil {
		newObjs := []*latest.OpenConfig{}
		for _, v := range config.Open {
			if v != nil {
				newObjs = append(newObjs, v)
			}
		}
		config.Open = newObjs
	}
}

// getVariables returns only the variables from the config
func getVariables(data map[string]interface{}) (map[string]interface{}, error) {
	retMap := map[string]interface{}{}
	err := util.Convert(data, &retMap)
	if err != nil {
		return nil, err
	}

	vars, ok := retMap["vars"]
	if !ok {
		return map[string]interface{}{
			"version": retMap["version"],
			"name":    retMap["name"],
		}, nil
	}

	return map[string]interface{}{
		"version": retMap["version"],
		"name":    retMap["name"],
		"vars":    vars,
	}, nil
}

// getProfiles loads a certain profile
func getProfiles(ctx context.Context, basePath string, data map[string]interface{}, profile string, profileChain *[]map[string]interface{}, depth int, update bool, log log.Logger) error {
	if depth > 50 {
		return fmt.Errorf("cannot load config with profile %s: max config loading depth reached. Seems like you have a profile cycle somewhere", profile)
	}

	// Convert config
	retMap := map[string]interface{}{}
	err := util.Convert(data, &retMap)
	if err != nil {
		return err
	}

	// Check if a profile is defined
	if profile == "" {
		return nil
	}

	// Convert to array
	profiles, ok := retMap["profiles"].([]interface{})
	if !ok {
		return errors.Errorf("Couldn't load profile '%s': no profiles found", profile)
	}

	// Search for config
	for i, profileMap := range profiles {
		profileConfig := &latest.ProfileConfig{}
		o, err := yaml.Marshal(profileMap)
		if err != nil {
			return err
		}
		decoder := yaml.NewDecoder(bytes.NewReader(o))
		decoder.KnownFields(true)
		err = decoder.Decode(profileConfig)
		if err != nil {
			return fmt.Errorf("error parsing profile at profiles[%d]: %v", i, err)
		}

		configMap, ok := profileMap.(map[string]interface{})
		if ok && profileConfig.Name == profile {
			// Add to profile chain
			*profileChain = append(*profileChain, configMap)

			// Get parents profiles
			if profileConfig.Parent != "" && len(profileConfig.Parents) > 0 {
				return errors.Errorf("parents and parent cannot be defined at the same time in profile %s. Please choose either one", profile)
			}

			// single parent
			if profileConfig.Parent != "" {
				return getProfiles(ctx, basePath, data, profileConfig.Parent, profileChain, depth+1, update, log)
			}

			// multiple parents
			if len(profileConfig.Parents) > 0 {
				for i := len(profileConfig.Parents) - 1; i >= 0; i-- {
					if profileConfig.Parents[i].Profile == "" {
						continue
					}

					if profileConfig.Parents[i].Source != nil {
						configPath, err := dependencyutil.DownloadDependency(ctx, basePath, profileConfig.Parents[i].Source, log)
						if err != nil {
							return err
						}

						fileContent, err := ioutil.ReadFile(configPath)
						if err != nil {
							return errors.Wrap(err, "read parent config")
						}

						rawMap := map[string]interface{}{}
						err = yaml.Unmarshal(fileContent, &rawMap)
						if err != nil {
							return err
						}

						err = getProfiles(ctx, filepath.Dir(configPath), rawMap, profileConfig.Parents[i].Profile, profileChain, depth+1, update, log)
						if err != nil {
							return errors.Wrapf(err, "load parent profile %s", profileConfig.Parents[i].Profile)
						}
					} else {
						err := getProfiles(ctx, basePath, data, profileConfig.Parents[i].Profile, profileChain, depth+1, update, log)
						if err != nil {
							return err
						}
					}
				}
			}

			return nil
		}
	}

	// Couldn't find config
	return errors.Errorf("Couldn't find profile '%s'", profile)
}

func getActivatedProfiles(data map[string]interface{}, resolver variable.Resolver) ([]string, error) {
	activatedProfiles := []string{}

	// Check if there are profiles
	if data["profiles"] == nil {
		return activatedProfiles, nil
	}

	// Convert to array
	profiles, ok := data["profiles"].([]interface{})
	if !ok {
		return activatedProfiles, errors.Errorf("Couldn't load profiles: no profiles found")
	}

	// Select which profiles are activated
	for i, profileMap := range profiles {
		profileConfig := &latest.ProfileConfig{}

		o, err := yaml.Marshal(profileMap)
		if err != nil {
			return activatedProfiles, err
		}

		decoder := yaml.NewDecoder(bytes.NewReader(o))
		decoder.KnownFields(true)
		err = decoder.Decode(profileConfig)
		if err != nil {
			return activatedProfiles, fmt.Errorf("error parsing profile at profiles[%d]: %v", i, err)
		}

		for _, activation := range profileConfig.Activation {
			activatedByEnv, err := matchEnvironment(activation.Environment)
			if err != nil {
				return activatedProfiles, errors.Wrap(err, "error activating profile with env")
			}

			activatedByVars, err := matchVars(activation.Vars, resolver)
			if err != nil {
				return activatedProfiles, errors.Wrap(err, "error activating profile with vars")
			}

			if activatedByEnv && activatedByVars {
				activatedProfiles = append(activatedProfiles, profileConfig.Name)
			}
		}
	}

	return activatedProfiles, nil
}

func matchEnvironment(env map[string]string) (bool, error) {
	for k, v := range env {
		match, err := regexp.MatchString(sanitizeMatchExpression(v), os.Getenv(k))
		if err != nil {
			return false, err
		}

		if !match {
			return false, nil
		}
	}

	return true, nil
}

func matchVars(activationVars map[string]string, resolver variable.Resolver) (bool, error) {
	for k, v := range activationVars {
		value, err := resolveVariableValue(k, resolver)
		if err != nil {
			return false, err
		}

		match, err := regexp.MatchString(sanitizeMatchExpression(v), value)
		if err != nil {
			return false, err
		} else if !match {
			return false, nil
		}
	}

	return true, nil
}

func filterProfileParents(profileParents []string) []string {
	return util.Filter(profileParents, func(oidx int, os string) bool {
		return !util.Contains(profileParents, func(iidx int, is string) bool {
			return os == is
		}, oidx+1)
	})
}

func sanitizeMatchExpression(expression string) string {
	exp := strings.TrimPrefix(expression, "^")
	exp = strings.TrimSuffix(exp, "$")
	exp = fmt.Sprintf("^%s$", exp)
	return exp
}

func resolveVariableValue(name string, resolver variable.Resolver) (string, error) {
	val, err := resolver.FillVariables(context.TODO(), "${"+name+"}")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", val), nil
}
