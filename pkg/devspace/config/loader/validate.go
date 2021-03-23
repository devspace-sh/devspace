package loader

import (
	jsonyaml "github.com/ghodss/yaml"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/helm/merge"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	k8sv1 "k8s.io/api/core/v1"
	"path/filepath"
)

// ValidInitialSyncStrategy checks if strategy is valid
func ValidInitialSyncStrategy(strategy latest.InitialSyncStrategy) bool {
	return strategy == "" ||
		strategy == latest.InitialSyncStrategyMirrorLocal ||
		strategy == latest.InitialSyncStrategyMirrorRemote ||
		strategy == latest.InitialSyncStrategyKeepAll ||
		strategy == latest.InitialSyncStrategyPreferLocal ||
		strategy == latest.InitialSyncStrategyPreferRemote ||
		strategy == latest.InitialSyncStrategyPreferNewest
}

func validate(config *latest.Config, log log.Logger) error {
	err := validateImages(config)
	if err != nil {
		return err
	}

	err = validateDev(config)
	if err != nil {
		return err
	}

	err = validateHooks(config)
	if err != nil {
		return err
	}

	err = validateDeployments(config)
	if err != nil {
		return err
	}

	err = validatePullSecrets(config)
	if err != nil {
		return err
	}

	err = validateCommands(config)
	if err != nil {
		return err
	}

	return nil
}

func validateCommands(config *latest.Config) error {
	for index, command := range config.Commands {
		if command.Name == "" {
			return errors.Errorf("commands[%d].name is required", index)
		}
		if command.Command == "" {
			return errors.Errorf("commands[%d].command is required", index)
		}
	}

	return nil
}

func validateHooks(config *latest.Config) error {
	for index, hookConfig := range config.Hooks {
		if hookConfig.Command == "" && hookConfig.Upload == nil && hookConfig.Download == nil && hookConfig.Logs == nil && hookConfig.Wait == nil {
			return errors.Errorf("hooks[%d].command, hooks[%d].logs, hooks[%d].wait, hooks[%d].download or hooks[%d].upload is required", index, index, index, index, index)
		}
		enabled := 0
		if hookConfig.Command != "" {
			enabled++
		}
		if hookConfig.Download != nil {
			enabled++
		}
		if hookConfig.Upload != nil {
			enabled++
		}
		if hookConfig.Logs != nil {
			enabled++
		}
		if hookConfig.Wait != nil {
			enabled++
		}
		if enabled > 1 {
			return errors.Errorf("you can only use one of hooks[%d].command, hooks[%d].logs, hooks[%d].wait, hooks[%d].upload and hooks[%d].download per hook", index, index, index, index, index)
		}
		if hookConfig.Upload != nil && hookConfig.Where.Container == nil {
			return errors.Errorf("hooks[%d].where.container is required if hooks[%d].upload is used", index, index)
		}
		if hookConfig.Download != nil && hookConfig.Where.Container == nil {
			return errors.Errorf("hooks[%d].where.container is required if hooks[%d].download is used", index, index)
		}
		if hookConfig.Logs != nil && hookConfig.Where.Container == nil {
			return errors.Errorf("hooks[%d].where.container is required if hooks[%d].logs is used", index, index)
		}
		if hookConfig.Wait != nil && hookConfig.Where.Container == nil {
			return errors.Errorf("hooks[%d].where.container is required if hooks[%d].wait is used", index, index)
		}
		if hookConfig.Wait != nil && hookConfig.Wait.Running == false && hookConfig.Wait.TerminatedWithCode == nil {
			return errors.Errorf("hooks[%d].wait.running or hooks[%d].wait.terminatedWithCode is required if hooks[%d].wait is used", index, index, index)
		}
	}

	return nil
}

func validateDeployments(config *latest.Config) error {
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

	return nil
}

func validatePullSecrets(config *latest.Config) error {
	for i, ps := range config.PullSecrets {
		if ps.Registry == "" {
			return errors.Errorf("pullSecrets[%d].registry: cannot be empty", i)
		}
	}

	return nil
}

func validateImages(config *latest.Config) error {
	// images lists all the image names in order to check for duplicates
	images := map[string]bool{}
	for imageConfigName, imageConf := range config.Images {
		if imageConfigName == "" {
			return errors.Errorf("images keys cannot be an empty string")
		}
		if imageConf == nil {
			return errors.Errorf("images.%s is empty and should at least contain an image name", imageConfigName)
		}
		if imageConf.Image == "" {
			return errors.Errorf("images.%s.image is required", imageConfigName)
		}
		if imageConf.Build != nil && imageConf.Build.Custom != nil && imageConf.Build.Custom.Command == "" && len(imageConf.Build.Custom.Commands) == 0 {
			return errors.Errorf("images.%s.build.custom.command or images.%s.build.custom.commands is required", imageConfigName, imageConfigName)
		}
		if images[imageConf.Image] {
			return errors.Errorf("multiple image definitions with the same image name are not allowed")
		}
		if imageConf.RebuildStrategy != latest.RebuildStrategyDefault && imageConf.RebuildStrategy != latest.RebuildStrategyAlways && imageConf.RebuildStrategy != latest.RebuildStrategyIgnoreContextChanges {
			return errors.Errorf("images.%s.rebuildStrategy %s is invalid. Please choose one of %v", imageConfigName, string(imageConf.RebuildStrategy), []latest.RebuildStrategy{latest.RebuildStrategyAlways, latest.RebuildStrategyIgnoreContextChanges})
		}
		if imageConf.Build != nil && imageConf.Build.Kaniko != nil && imageConf.Build.Kaniko.EnvFrom != nil {
			for _, v := range imageConf.Build.Kaniko.EnvFrom {
				o, err := yaml.Marshal(v)
				if err != nil {
					return errors.Errorf("images.%s.build.kaniko.envFrom is invalid: %v", imageConfigName, err)
				}

				err = jsonyaml.Unmarshal(o, &k8sv1.EnvVarSource{})
				if err != nil {
					return errors.Errorf("images.%s.build.kaniko.envFrom is invalid: %v", imageConfigName, err)
				}
			}
		}
		images[imageConf.Image] = true
	}

	return nil
}

func validateDev(config *latest.Config) error {
	if config.Dev != nil {
		if config.Dev.Ports != nil {
			for index, port := range config.Dev.Ports {
				// Validate imageName and label selector
				if port.ImageName == "" && len(port.LabelSelector) == 0 {
					return errors.Errorf("Error in config: imageName and label selector are nil in ports config at index %d", index)
				} else if port.ImageName != "" && findImageName(config, port.ImageName) == false {
					return errors.Errorf("Error in config: dev.ports[%d].imageName '%s' couldn't be found. Please make sure the image name exists under 'images'", index, port.ImageName)
				}

				if len(port.PortMappings) == 0 && len(port.PortMappingsReverse) == 0 {
					return errors.Errorf("Error in config: portMappings is empty in port config at index %d", index)
				}
			}
		}

		if config.Dev.Sync != nil {
			for index, sync := range config.Dev.Sync {
				// Validate imageName and label selector
				if sync.ImageName == "" && len(sync.LabelSelector) == 0 {
					return errors.Errorf("Error in config: imageName and label selector are nil in sync config at index %d", index)
				} else if sync.ImageName != "" && findImageName(config, sync.ImageName) == false {
					return errors.Errorf("Error in config: dev.sync[%d].imageName '%s' couldn't be found. Please make sure the image name exists under 'images'", index, sync.ImageName)
				}

				// Validate initial sync strategy
				if ValidInitialSyncStrategy(sync.InitialSync) == false {
					return errors.Errorf("Error in config: sync.initialSync is not valid '%s' at index %d", sync.InitialSync, index)
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

	return nil
}

func findImageName(config *latest.Config, imageName string) bool {
	if config.Images == nil {
		return false
	}

	return config.Images[imageName] != nil
}
