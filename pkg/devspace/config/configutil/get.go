package configutil

import (
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
var getConfigOnceMutex sync.Mutex

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

// ResetConfig resets the current config
func ResetConfig() {
	getConfigOnceMutex.Lock()
	defer getConfigOnceMutex.Unlock()

	getConfigOnce = sync.Once{}
}

// InitConfig initializes the config objects
func InitConfig() *latest.Config {
	getConfigOnceMutex.Lock()
	defer getConfigOnceMutex.Unlock()

	getConfigOnce.Do(func() {
		config = latest.New().(*latest.Config)
	})

	return config
}

// GetBaseConfig returns the config
func GetBaseConfig(overrideKubeContext string) (*latest.Config, error) {
	return loadConfigOnce(overrideKubeContext, "", false)
}

// GetConfig returns the config merged with all potential overwrite files
func GetConfig(overrideKubeContext, profile string) (*latest.Config, error) {
	return loadConfigOnce(overrideKubeContext, profile, true)
}

// GetConfigFromPath loads the config from a given base path
func GetConfigFromPath(generatedConfig *generated.Config, basePath, kubeContext, profile string, log log.Logger) (*latest.Config, error) {
	configPath := filepath.Join(basePath, constants.DefaultConfigPath)

	// Check devspace.yaml
	_, err := os.Stat(configPath)
	if err != nil {
		// Check for legacy devspace-configs.yaml
		_, err = os.Stat(filepath.Join(basePath, constants.DefaultConfigsPath))
		if err == nil {
			return nil, errors.Errorf("devspace-configs.yaml is not supported anymore in devspace v4. Please use 'profiles' in 'devspace.yaml' instead")
		}

		return nil, errors.Errorf("Couldn't find '%s': %v", err)
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

	loadedConfig, err := ParseConfig(generatedConfig, rawMap, kubeContext, profile, log)
	if err != nil {
		return nil, err
	}

	// Now we validate the config
	err = validate(loadedConfig)
	if err != nil {
		return nil, err
	}

	err = ApplyReplace(loadedConfig)
	if err != nil {
		return nil, err
	}

	// Apply patches
	loadedConfig, err = ApplyPatches(loadedConfig)
	if err != nil {
		return nil, err
	}

	err = validate(loadedConfig)
	if err != nil {
		return nil, err
	}

	return loadedConfig, nil
}

// loadConfigOnce loads the config globally once
func loadConfigOnce(kubeContext, profile string, allowProfile bool) (*latest.Config, error) {
	getConfigOnceMutex.Lock()
	defer getConfigOnceMutex.Unlock()

	var retError error
	getConfigOnce.Do(func() {
		// Get generated config
		generatedConfig, err := generated.LoadConfig(profile)
		if err != nil {
			retError = err
			return
		}

		// Check if we should load a specific config
		if allowProfile && generatedConfig.ActiveProfile != "" && profile == "" {
			profile = generatedConfig.ActiveProfile
		} else if !allowProfile {
			profile = ""
		}

		// Load base config
		config, err = GetConfigFromPath(generatedConfig, ".", kubeContext, profile, log.GetInstance())
		if err != nil {
			retError = err
			return
		}

		// Save generated config
		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			retError = err
			return
		}
	})

	return config, retError
}

func validate(config *latest.Config) error {
	if config.Dev != nil {
		if config.Dev.Ports != nil {
			for index, port := range config.Dev.Ports {
				if port.ImageName == "" && port.LabelSelector == nil {
					return errors.Errorf("Error in config: imageName and label selector are nil in port config at index %d", index)
				}
				if port.PortMappings == nil {
					return errors.Errorf("Error in config: portMappings is empty in port config at index %d", index)
				}
			}
		}

		if config.Dev.Sync != nil {
			for index, sync := range config.Dev.Sync {
				if sync.ImageName == "" && sync.LabelSelector == nil {
					return errors.Errorf("Error in config: imageName and label selector are nil in sync config at index %d", index)
				}
			}
		}

		if config.Dev.Interactive != nil {
			for index, imageConf := range config.Dev.Interactive.Images {
				if imageConf.Name == "" {
					return errors.Errorf("Error in config: Unnamed interactive image config at index %d", index)
				}
			}
		}
	}

	if config.Hooks != nil {
		for index, hookConfig := range config.Hooks {
			if hookConfig.Command == "" {
				return errors.Errorf("hooks[%d].command is required", index)
			}
		}
	}

	if config.Images != nil {
		for imageConfigName, imageConf := range config.Images {
			if imageConf.Build != nil && imageConf.Build.Custom != nil && imageConf.Build.Custom.Command == "" {
				return errors.Errorf("images.%s.build.custom.command is required", imageConfigName)
			}
		}
	}

	if config.Deployments != nil {
		for index, deployConfig := range config.Deployments {
			if deployConfig.Name == "" {
				return errors.Errorf("deployments[%d].name is required", index)
			}
			if deployConfig.Helm == nil && deployConfig.Kubectl == nil && deployConfig.Component == nil {
				return errors.Errorf("Please specify either component, helm or kubectl as deployment type in deployment %s", deployConfig.Name)
			}
			if deployConfig.Helm != nil && (deployConfig.Helm.Chart == nil || deployConfig.Helm.Chart.Name == "") {
				return errors.Errorf("deployments[%d].helm.chart and deployments[%d].helm.chart.name is required", index, index)
			}
			if deployConfig.Kubectl != nil && deployConfig.Kubectl.Manifests == nil {
				return errors.Errorf("deployments[%d].kubectl.manifests is required", index)
			}
		}
	}

	if len(config.Profiles) > 0 {
		for idx, profile := range config.Profiles {
			if profile.Name == "" {
				return errors.Errorf("profiles.%d.name is missing", idx)
			}

			for patchIdx, patch := range profile.Patches {
				if patch.Operation == "" {
					return errors.Errorf("profiles.%s.patches.%d.op is missing", profile.Name, patchIdx)
				}
				if patch.Path == "" {
					return errors.Errorf("profiles.%s.patches.%d.path is missing", profile.Name, patchIdx)
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
