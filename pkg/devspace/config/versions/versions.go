package versions

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1alpha1"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1alpha2"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1alpha3"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1alpha4"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta1"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta2"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta3"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta4"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta5"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta6"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta7"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta8"
	dependencyutil "github.com/loft-sh/devspace/pkg/devspace/dependency/util"
	"github.com/loft-sh/devspace/pkg/util/log"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type loader struct {
	New config.New
}

var versionLoader = map[string]*loader{
	v1alpha1.Version: &loader{New: v1alpha1.New},
	v1alpha2.Version: &loader{New: v1alpha2.New},
	v1alpha3.Version: &loader{New: v1alpha3.New},
	v1alpha4.Version: &loader{New: v1alpha4.New},
	v1beta1.Version:  &loader{New: v1beta1.New},
	v1beta2.Version:  &loader{New: v1beta2.New},
	v1beta3.Version:  &loader{New: v1beta3.New},
	v1beta4.Version:  &loader{New: v1beta4.New},
	v1beta5.Version:  &loader{New: v1beta5.New},
	v1beta6.Version:  &loader{New: v1beta6.New},
	v1beta7.Version:  &loader{New: v1beta7.New},
	v1beta8.Version:  &loader{New: v1beta8.New},
	latest.Version:   &loader{New: latest.New},
}

// ParseProfile loads the base config & a certain profile
func ParseProfile(basePath string, data map[interface{}]interface{}, profile string, profileParents []string, update bool, log log.Logger) ([]map[interface{}]interface{}, error) {
	profiles := []map[interface{}]interface{}{}
	if len(profileParents) > 0 && profile == "" {
		profile = profileParents[len(profileParents)-1]
		profileParents = profileParents[:len(profileParents)-1]
	}

	err := getProfiles(basePath, data, profile, &profiles, 1, update, log)
	if err != nil {
		return nil, err
	}

	// check if there are profile parents
	for i := len(profileParents) - 1; i >= 0; i-- {
		err := getProfiles(basePath, data, profileParents[i], &profiles, 1, update, log)
		if err != nil {
			return nil, err
		}
	}

	return profiles, nil
}

// ParseCommands parses only the commands from the config
func ParseCommands(data map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	return getCommands(data)
}

// ParseVariables parses only the variables from the config
func ParseVariables(data map[interface{}]interface{}, log log.Logger) ([]*latest.Variable, error) {
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
func Parse(data map[interface{}]interface{}, log log.Logger) (*latest.Config, error) {
	version, ok := data["version"].(string)
	if ok == false {
		return nil, errors.Errorf("Version is missing in devspace.yaml")
	}

	loader, ok := versionLoader[version]
	if ok == false {
		return nil, errors.Errorf("Unrecognized config version %s. Please upgrade devspace with `devspace upgrade`", version)
	}

	versionLoadFunc := loader.New

	// Load config strict
	latestConfig := versionLoadFunc()
	out, err := yaml.Marshal(data)
	if err != nil {
		return nil, err
	}
	err = yaml.UnmarshalStrict(out, latestConfig)
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
	if ok == false {
		return nil, errors.Errorf("Error converting config, latest config is not the latest version")
	}

	// Update version to latest
	latestConfigConverted.Version = latest.Version

	return latestConfigConverted, nil
}

// getVariables returns only the variables from the config
func getVariables(data map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	retMap := map[interface{}]interface{}{}
	err := util.Convert(data, &retMap)
	if err != nil {
		return nil, err
	}

	vars, ok := retMap["vars"]
	if !ok {
		return map[interface{}]interface{}{
			"version": retMap["version"],
		}, nil
	}

	return map[interface{}]interface{}{
		"version": retMap["version"],
		"vars":    vars,
	}, nil
}

// getCommands returns only the commands from the config
func getCommands(data map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	retMap := map[interface{}]interface{}{}
	err := util.Convert(data, &retMap)
	if err != nil {
		return nil, err
	}

	commands, ok := retMap["commands"]
	if !ok {
		return map[interface{}]interface{}{
			"version": retMap["version"],
		}, nil
	}

	return map[interface{}]interface{}{
		"version":  retMap["version"],
		"commands": commands,
	}, nil
}

// getProfiles loads a certain profile
func getProfiles(basePath string, data map[interface{}]interface{}, profile string, profileChain *[]map[interface{}]interface{}, depth int, update bool, log log.Logger) error {
	if depth > 50 {
		return fmt.Errorf("cannot load config with profile %s: max config loading depth reached. Seems like you have a profile cycle somewhere", profile)
	}

	// Convert config
	retMap := map[interface{}]interface{}{}
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
		err = yaml.UnmarshalStrict(o, profileConfig)
		if err != nil {
			return fmt.Errorf("error parsing profile at profiles[%d]: %v", i, err)
		}

		configMap, ok := profileMap.(map[interface{}]interface{})
		if ok && profileConfig.Name == profile {
			// Add to profile chain
			*profileChain = append(*profileChain, configMap)

			// Get parents profiles
			if profileConfig.Parent != "" && len(profileConfig.Parents) > 0 {
				return errors.Errorf("parents and parent cannot be defined at the same time in profile %s. Please choose either one", profile)
			}

			// single parent
			if profileConfig.Parent != "" {
				return getProfiles(basePath, data, profileConfig.Parent, profileChain, depth+1, update, log)
			}

			// multiple parents
			if len(profileConfig.Parents) > 0 {
				for i := len(profileConfig.Parents) - 1; i >= 0; i-- {
					if profileConfig.Parents[i].Profile == "" {
						continue
					}

					if profileConfig.Parents[i].Source != nil {
						_, localPath, err := dependencyutil.DownloadDependency(basePath, profileConfig.Parents[i].Source, profileConfig.Parents[i].Profile, nil, update, log)
						if err != nil {
							return err
						}

						configPath := filepath.Join(localPath, constants.DefaultConfigPath)
						if profileConfig.Parents[i].Source.ConfigName != "" {
							configPath = filepath.Join(localPath, profileConfig.Parents[i].Source.ConfigName)
						}

						fileContent, err := ioutil.ReadFile(configPath)
						if err != nil {
							return errors.Wrap(err, "read parent config")
						}

						rawMap := map[interface{}]interface{}{}
						err = yaml.Unmarshal(fileContent, &rawMap)
						if err != nil {
							return err
						}

						err = getProfiles(localPath, rawMap, profileConfig.Parents[i].Profile, profileChain, depth+1, update, log)
						if err != nil {
							return errors.Wrapf(err, "load parent profile %s", profileConfig.Parents[i].Profile)
						}
					} else {
						err := getProfiles(basePath, data, profileConfig.Parents[i].Profile, profileChain, depth+1, update, log)
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
