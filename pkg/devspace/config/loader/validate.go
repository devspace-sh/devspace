package loader

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer/helm/merge"
	"github.com/devspace-cloud/devspace/pkg/util/yamlutil"
)

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

		// if config.Dev.Sync != nil {
		// 	for index, sync := range config.Dev.Sync {
		// 		if sync.ImageName == "" && sync.LabelSelector == nil {
		// 			return errors.Errorf("Error in config: imageName and label selector are nil in sync config at index %d", index)
		// 		}
		// 	}
		// }

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
			if imageConf.Build != nil && imageConf.Build.Custom != nil && imageConf.Build.Custom.Command == "" {
				return errors.Errorf("images.%s.build.custom.command is required", imageConfigName)
			}
			if imageConf.Image == "" {
				return fmt.Errorf("images.%s.image is required", imageConfigName)
			}
			if images[imageConf.Image] {
				return errors.Errorf("multiple image definitions with the same image name are not allowed")
			}
			images[imageConf.Image] = true
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
