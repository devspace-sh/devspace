package versions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta11"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"

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

var VersionLoader = map[string]*loader{
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
func ParseProfile(ctx context.Context, basePath string, data map[string]interface{}, profiles []string, update bool, disableProfileActivation bool, resolver variable.Resolver, log log.Logger) ([]*latest.ProfileConfig, error) {
	parsedProfiles := []*latest.ProfileConfig{}

	// auto activated root level profiles
	activatedProfiles := []string{}
	if !disableProfileActivation {
		var err error
		activatedProfiles, err = getActivatedProfiles(ctx, data, resolver, log)
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

// Get parses only the key from the config
func Get(data map[string]interface{}, keys ...string) (map[string]interface{}, error) {
	retMap := map[string]interface{}{}
	err := util.Convert(data, &retMap)
	if err != nil {
		return nil, err
	}

	retConfig := map[string]interface{}{
		"version": retMap["version"],
		"name":    retMap["name"],
	}
	for _, key := range keys {
		keyData, ok := retMap[key]
		if ok {
			retConfig[key] = keyData
		}
	}

	return retConfig, nil
}

// ParseVariables parses only the variables from the config
func ParseVariables(data map[string]interface{}, log log.Logger) (map[string]*latest.Variable, error) {
	strippedData, err := Get(data, "vars")
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

	loader, ok := VersionLoader[version]
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
	err = yamlutil.UnmarshalStrict(out, latestConfig)
	if err != nil {
		return nil, errors.Errorf("error loading config: %v", err)
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
	err = adjustConfig(latestConfigConverted)
	if err != nil {
		return nil, err
	}

	// validate config
	err = Validate(latestConfigConverted)
	if err != nil {
		return nil, err
	}

	return latestConfigConverted, nil
}

// getProfiles loads a certain profile
func getProfiles(ctx context.Context, basePath string, data map[string]interface{}, profile string, profileChain *[]*latest.ProfileConfig, depth int, update bool, log log.Logger) error {
	if depth > 50 {
		return fmt.Errorf("cannot load config with profile %s: max config loading depth reached. Seems like you have a profile cycle somewhere", profile)
	}

	// Check if a profile is defined
	if profile == "" {
		return nil
	}

	// get the profiles and parse them
	profilesData, err := Get(data, "profiles")
	if err != nil {
		return err
	}
	profiles, err := Parse(profilesData, log)
	if err != nil {
		return err
	}

	// Search for config
	for _, profileConfig := range profiles.Profiles {
		if profileConfig.Name == profile {
			// Add to profile chain
			*profileChain = append(*profileChain, profileConfig)

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

						fileContent, err := os.ReadFile(configPath)
						if err != nil {
							return errors.Wrap(err, "read parent config")
						}

						rawMap := map[string]interface{}{}
						err = yamlutil.Unmarshal(fileContent, &rawMap)
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

func getActivatedProfiles(ctx context.Context, data map[string]interface{}, resolver variable.Resolver, log log.Logger) ([]string, error) {
	activatedProfiles := []string{}

	// Check if there are profiles
	if data["profiles"] == nil {
		return activatedProfiles, nil
	}

	// get the profiles and parse them
	profilesData, err := Get(data, "profiles")
	if err != nil {
		return nil, err
	}
	profiles, err := Parse(profilesData, log)
	if err != nil {
		return nil, err
	}

	// Select which profiles are activated
	for _, profileConfig := range profiles.Profiles {
		for _, activation := range profileConfig.Activation {
			activatedByEnv, err := matchEnvironment(activation.Environment)
			if err != nil {
				return activatedProfiles, errors.Wrap(err, "error activating profile with env")
			}

			activatedByVars, err := matchVars(ctx, activation.Vars, resolver)
			if err != nil {
				return activatedProfiles, errors.Wrap(err, "error activating profile with vars")
			}

			if activatedByEnv && activatedByVars {
				log.Debugf("profile %s was automatically activated", profileConfig.Name)
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

func matchVars(ctx context.Context, activationVars map[string]string, resolver variable.Resolver) (bool, error) {
	for k, v := range activationVars {
		value, err := resolveVariableValue(ctx, k, resolver)
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

func resolveVariableValue(ctx context.Context, name string, resolver variable.Resolver) (string, error) {
	val, err := resolver.FillVariables(ctx, "${"+name+"}", true)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", val), nil
}
