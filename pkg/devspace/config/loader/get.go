package loader

import (
	"fmt"
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
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer/helm/merge"
	"github.com/devspace-cloud/devspace/pkg/util/yamlutil"
)

// ConfigLoader is the base interface for the main config loader
type ConfigLoader interface {
	New() *latest.Config
	Exists() bool

	Load() (*latest.Config, error)
	LoadFromPath(generatedConfig *generated.Config, path string) (*latest.Config, error)
	LoadRaw(path string) (map[interface{}]interface{}, error)
	LoadWithoutProfile() (*latest.Config, error)

	ParseCommands(generatedConfig *generated.Config, data map[interface{}]interface{}) ([]*latest.CommandConfig, error)

	Generated() (*generated.Config, error)
	SaveGenerated(generatedConfig *generated.Config) error

	Save(config *latest.Config) error
	RestoreVars(config *latest.Config) (*latest.Config, error)
	SetDevSpaceRoot() (bool, error)
}

type configLoader struct {
	generatedLoader generated.ConfigLoader
	generatedConfig *generated.Config

	options *ConfigOptions
	log     log.Logger
}

// NewConfigLoader creates a new config loader with the given options
func NewConfigLoader(options *ConfigOptions, log log.Logger) ConfigLoader {
	if options == nil {
		options = &ConfigOptions{}
	}

	// Set loaded vars for this
	options.LoadedVars = make(map[string]string)

	return &configLoader{
		generatedLoader: generated.NewConfigLoader(options.Profile),
		options:         options,
		log:             log,
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
func (l *configLoader) SaveGenerated(generatedConfig *generated.Config) error {
	return l.generatedLoader.Save(generatedConfig)
}

// LoadGenerated loads the generated config
func (l *configLoader) LoadGeneratedFromPath(path string) (*generated.Config, error) {
	return l.generatedLoader.LoadFromPath(path)
}

// Exists checks whether the yaml file for the config exists or the configs.yaml exists
func (l *configLoader) Exists() bool {
	return configExistsInPath(".")
}

// configExistsInPath checks wheter a devspace configuration exists at a certain path
func configExistsInPath(path string) bool {
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

// New initializes a new config object
func (l *configLoader) New() *latest.Config {
	return latest.New().(*latest.Config)
}

// ConfigOptions defines options to load the config
type ConfigOptions struct {
	Profile     string
	KubeContext string

	LoadedVars map[string]string
	Vars       []string
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
func (l *configLoader) LoadRaw(configPath string) (map[interface{}]interface{}, error) {
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

// LoadPath loads the config from a given base path
func (l *configLoader) LoadFromPath(generatedConfig *generated.Config, configPath string) (*latest.Config, error) {
	// Check devspace.yaml
	_, err := os.Stat(configPath)
	if err != nil {
		return nil, errors.Errorf("Couldn't find '%s': %v", configPath, err)
	}

	rawMap, err := l.LoadRaw(configPath)
	if err != nil {
		return nil, err
	}

	loadedConfig, err := l.parseConfig(generatedConfig, rawMap)
	if err != nil {
		return nil, err
	}

	// Now we validate the config
	err = validate(loadedConfig)
	if err != nil {
		return nil, err
	}

	return loadedConfig, nil
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

	// Load base config
	config, err := l.LoadFromPath(generatedConfig, constants.DefaultConfigPath)
	if err != nil {
		return nil, err
	}

	// Save generated config
	err = l.generatedLoader.Save(generatedConfig)
	if err != nil {
		return nil, err
	}

	return config, nil
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

	if config.Commands != nil {
		for index, command := range config.Commands {
			if command.Name == "" {
				return errors.Errorf("commands[%d].name is required", index)
			}
			if command.Command == "" {
				return errors.Errorf("commands[%d].command is required", index)
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
			if imageConfigName == "" {
				return errors.Errorf("images keys cannot be an empty string")
			}
			if imageConf.Image == "" {
				return errors.Errorf("images.%s.image is required", imageConfigName)
			}
			if imageConf.Build != nil && imageConf.Build.Custom != nil && imageConf.Build.Custom.Command == "" {
				return errors.Errorf("images.%s.build.custom.command is required", imageConfigName)
			}
			if imageConf.Image == "" {
				return fmt.Errorf("images.%s.image is required", imageConfigName)
			}
		}
	}

	if config.Deployments != nil {
		for index, deployConfig := range config.Deployments {
			if deployConfig.Name == "" {
				return errors.Errorf("deployments[%d].name is required", index)
			}
			if deployConfig.Helm == nil && deployConfig.Kubectl == nil {
				return errors.Errorf("Please specify either helm or kubectl as deployment type in deployment %s", deployConfig.Name)
			}
			if deployConfig.Helm != nil && (deployConfig.Helm.Chart == nil || deployConfig.Helm.Chart.Name == "") && (deployConfig.Helm.ComponentChart == nil || *deployConfig.Helm.ComponentChart == false) {
				return errors.Errorf("deployments[%d].helm.chart and deployments[%d].helm.chart.name or deployments[%d].helm.componentChart is required", index, index, index)
			}
			if deployConfig.Kubectl != nil && deployConfig.Kubectl.Manifests == nil {
				return errors.Errorf("deployments[%d].kubectl.manifests is required", index)
			}
			if deployConfig.Helm != nil && deployConfig.Helm.ComponentChart != nil && *deployConfig.Helm.ComponentChart == true {
				// Load override values from path
				overwriteValues := map[interface{}]interface{}{}
				if deployConfig.Helm.ValuesFiles != nil {
					for _, overridePath := range deployConfig.Helm.ValuesFiles {
						overwriteValuesPath, err := filepath.Abs(overridePath)
						if err != nil {
							return errors.Errorf("deployments[%d].helm.valuesFiles: Error retrieving absolute path from %s: %v", index, overridePath, err)
						}

						overwriteValuesFromPath := map[interface{}]interface{}{}
						err = yamlutil.ReadYamlFromFile(overwriteValuesPath, overwriteValuesFromPath)
						if err == nil {
							merge.Values(overwriteValues).MergeInto(overwriteValuesFromPath)
						}
					}
				}

				// Load override values from data and merge them
				if deployConfig.Helm.Values != nil {
					merge.Values(overwriteValues).MergeInto(deployConfig.Helm.Values)
				}

				bytes, err := yaml.Marshal(overwriteValues)
				if err != nil {
					return errors.Errorf("deployments[%d].helm: Error marshaling overwrite values: %v", index, err)
				}

				componentValues := &latest.ComponentConfig{}
				err = yaml.UnmarshalStrict(bytes, componentValues)
				if err != nil {
					return errors.Errorf("deployments[%d].helm.componentChart: component values are incorrect: %v", index, err)
				}
			}
		}
	}

	return nil
}

// SetDevSpaceRoot checks the current directory and all parent directories for a .devspace folder with a config and sets the current working directory accordingly
func (l *configLoader) SetDevSpaceRoot() (bool, error) {
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
