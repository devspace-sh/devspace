package configutil

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
)

// Global config vars
var config *latest.Config // merged config

// Thread-safety helper
var getConfigOnce sync.Once
var validateOnce sync.Once

// ConfigExists checks whether the yaml file for the config exists or the configs.yaml exists
func ConfigExists() bool {
	return configExistsInPath(".")
}

// configExistsInPath checks wheter a devspace configuration exists at a certain path
func configExistsInPath(path string) bool {
	// Needed for testing
	if config != nil {
		return true
	}

	// Check devspace.yaml
	_, err := os.Stat(filepath.Join(path, constants.DefaultConfigPath))
	if err == nil {
		return true
	}

	// Check devspace-configs.yaml
	_, err = os.Stat(filepath.Join(path, constants.DefaultConfigsPath))
	if err == nil {
		return true
	}

	return false // Normal config file found
}

// InitConfig initializes the config objects
func InitConfig() *latest.Config {
	getConfigOnce.Do(func() {
		config = latest.New().(*latest.Config)
	})

	return config
}

// GetBaseConfig returns the config
func GetBaseConfig(ctx context.Context) *latest.Config {
	GetConfigOnce(ctx, false)
	ValidateOnce()

	return config
}

// GetConfig returns the config merged with all potential overwrite files
func GetConfig(ctx context.Context) *latest.Config {
	GetConfigOnce(ctx, true)
	ValidateOnce()

	return config
}

func loadConfigFromPath(ctx context.Context, generatedConfig *generated.Config, basePath string, log log.Logger) (*latest.Config, error) {
	configPath := filepath.Join(basePath, constants.DefaultConfigPath)

	// Check devspace.yaml
	_, err := os.Stat(configPath)
	if err != nil {
		// Check for legacy devspace-configs.yaml
		_, err = os.Stat(filepath.Join(basePath, constants.DefaultConfigsPath))
		if err == nil {
			return nil, fmt.Errorf("devspace-configs.yaml is not supported anymore in devspace v4. Please use the new config option 'configs' in 'devspace.yaml'")
		}

		return nil, fmt.Errorf("Couldn't find '%s': %v", err)
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

	loadedConfig, err := ParseConfig(ctx, generatedConfig, rawMap)
	if err != nil {
		return nil, err
	}

	err = ApplyReplace(loadedConfig)
	if err != nil {
		return nil, err
	}

	// Apply patches
	return ApplyPatches(loadedConfig)
}

// GetConfigFromPath loads the config from a given base path
func GetConfigFromPath(ctx context.Context, generatedConfig *generated.Config, basePath string, log log.Logger) (*latest.Config, error) {
	config, err := loadConfigFromPath(ctx, generatedConfig, basePath, log)
	if err != nil {
		return nil, err
	}

	err = validate(config)
	if err != nil {
		return nil, fmt.Errorf("Error validating config in %s: %v", basePath, err)
	}

	return config, nil
}

// GetConfigOnce loads the config globally once
func GetConfigOnce(ctx context.Context, allowProfile bool) *latest.Config {
	getConfigOnce.Do(func() {
		// Get generated config
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Panicf("Error loading %s: %v", generated.ConfigPath, err)
		}

		// Check if we should load a specific config
		if allowProfile && generatedConfig.ActiveProfile != "" && ctx.Value(constants.ProfileContextKey) == nil {
			ctx = context.WithValue(ctx, constants.ProfileContextKey, generatedConfig.ActiveProfile)
		} else if !allowProfile {
			ctx = context.WithValue(ctx, constants.ProfileContextKey, nil)
		}

		// Load base config
		config, err = loadConfigFromPath(ctx, generatedConfig, ".", log.GetInstance())
		if err != nil {
			log.Fatal(err)
		}

		// Save generated config
		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatalf("Couldn't save generated config: %v", err)
		}
	})

	return config
}

// ValidateOnce ensures that specific values are set in the config
func ValidateOnce() {
	validateOnce.Do(func() {
		err := validate(config)
		if err != nil {
			log.Fatal(err)
		}
	})
}

func validate(config *latest.Config) error {
	if config.Dev != nil {
		if config.Dev.Selectors != nil {
			for index, selectorConfig := range config.Dev.Selectors {
				if selectorConfig.Name == "" {
					return fmt.Errorf("Error in config: Unnamed selector at index %d", index)
				}
			}
		}

		if config.Dev.Ports != nil {
			for index, port := range config.Dev.Ports {
				if port.Selector == "" && port.LabelSelector == nil {
					return fmt.Errorf("Error in config: selector and label selector are nil in port config at index %d", index)
				}
				if port.PortMappings == nil {
					return fmt.Errorf("Error in config: portMappings is empty in port config at index %d", index)
				}
			}
		}

		if config.Dev.Sync != nil {
			for index, sync := range config.Dev.Sync {
				if sync.Selector == "" && sync.LabelSelector == nil {
					return fmt.Errorf("Error in config: selector and label selector are nil in sync config at index %d", index)
				}
			}
		}

		if config.Dev.OverrideImages != nil {
			for index, overrideImageConfig := range config.Dev.OverrideImages {
				if overrideImageConfig.Name == "" {
					return fmt.Errorf("Error in config: Unnamed override image config at index %d", index)
				}
			}
		}
	}

	if config.Hooks != nil {
		for index, hookConfig := range config.Hooks {
			if hookConfig.Command == "" {
				return fmt.Errorf("hooks[%d].command is required", index)
			}
		}
	}

	if config.Images != nil {
		for imageConfigName, imageConf := range config.Images {
			if imageConf.Build != nil && imageConf.Build.Custom != nil && imageConf.Build.Custom.Command == "" {
				return fmt.Errorf("images.%s.build.custom.command is required", imageConfigName)
			}
		}
	}

	if config.Deployments != nil {
		for index, deployConfig := range config.Deployments {
			if deployConfig.Name == "" {
				return fmt.Errorf("deployments[%d].name is required", index)
			}
			if deployConfig.Helm == nil && deployConfig.Kubectl == nil && deployConfig.Component == nil {
				return fmt.Errorf("Please specify either component, helm or kubectl as deployment type in deployment %s", deployConfig.Name)
			}
			if deployConfig.Helm != nil && (deployConfig.Helm.Chart == nil || deployConfig.Helm.Chart.Name == "") {
				return fmt.Errorf("deployments[%d].helm.chart and deployments[%d].helm.chart.name is required", index, index)
			}
			if deployConfig.Kubectl != nil && deployConfig.Kubectl.Manifests == nil {
				return fmt.Errorf("deployments[%d].kubectl.manifests is required", index)
			}
		}
	}

	if len(config.Profiles) > 0 {
		for idx, profile := range config.Profiles {
			if profile.Name == "" {
				return fmt.Errorf("profiles.%d.name is missing", idx)
			}

			for patchIdx, patch := range profile.Patches {
				if patch.Operation == "" {
					return fmt.Errorf("profiles.%s.patches.%d.op is missing", profile.Name, patchIdx)
				}
				if patch.Path == "" {
					return fmt.Errorf("profiles.%s.patches.%d.path is missing", profile.Name, patchIdx)
				}
			}
		}
	}

	return nil
}

// SetDevSpaceRoot checks the current directory and all parent directories for a .devspace folder with a config and sets the current working directory accordingly
func SetDevSpaceRoot() (bool, error) {
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
			configExists := configExistsInPath(cwd)
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

// GetSelector returns the service referenced by serviceName
func GetSelector(config *latest.Config, selectorName string) (*latest.SelectorConfig, error) {
	if config.Dev.Selectors != nil {
		for _, selector := range config.Dev.Selectors {
			if selector.Name == selectorName {
				return selector, nil
			}
		}
	}

	return nil, errors.New("Unable to find selector: " + selectorName)
}
