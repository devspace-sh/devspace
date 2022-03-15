package loader

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/imageselector"
	"strings"

	jsonyaml "github.com/ghodss/yaml"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	k8sv1 "k8s.io/api/core/v1"
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

// ValidContainerArch checks if the target container arch is valid
func ValidContainerArch(arch latest.ContainerArchitecture) bool {
	return arch == "" ||
		arch == latest.ContainerArchitectureAmd64 ||
		arch == latest.ContainerArchitectureArm64
}

func validate(config *latest.Config, log log.Logger) error {
	err := validateRequire(config)
	if err != nil {
		return err
	}

	err = validateImages(config)
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

	err = validateDependencies(config)
	if err != nil {
		return err
	}

	return nil
}

func validateVars(vars []*latest.Variable) error {
	for i, v := range vars {
		if v.Name == "" {
			return fmt.Errorf("vars[*].name has to be specified")
		}

		// make sure is unique
		for j, v2 := range vars {
			if i != j && v.Name == v2.Name {
				return fmt.Errorf("multiple definitions for variable %s found", v.Name)
			}
		}
	}

	return nil
}

func validateRequire(config *latest.Config) error {
	for index, plugin := range config.Require.Plugins {
		if plugin.Name == "" {
			return errors.Errorf("require.plugins[%d].name is required", index)
		}
		if plugin.Version == "" {
			return errors.Errorf("require.plugins[%d].version is required", index)
		}
	}

	for index, command := range config.Require.Commands {
		if command.Name == "" {
			return errors.Errorf("require.commands[%d].name is required", index)
		}
		if command.Version == "" {
			return errors.Errorf("require.commands[%d].version is required", index)
		}
	}

	return nil
}

func validateDependencies(config *latest.Config) error {
	for index, dep := range config.Dependencies {
		if dep.Name == "" {
			return errors.Errorf("dependencies[%d].name is required", index)
		}
		if strings.Contains(dep.Name, ".") {
			return errors.Errorf("dependencies[%d].name cannot contain a '.'", index)
		}
		if !isDependencyNameUnique(dep.Name, config.Dependencies) {
			return errors.Errorf("dependencies[%d].name has to be unique", index)
		}
		if dep.Source == nil {
			return errors.Errorf("dependencies[%d].source is required", index)
		}
		if dep.Source.Git == "" && dep.Source.Path == "" {
			return errors.Errorf("dependencies[%d].source.git or dependencies[%d].source.path is required", index, index)
		}
		if len(dep.Profiles) > 0 && (dep.Profile != "" || len(dep.ProfileParents) > 0) {
			return errors.Errorf("dependencies[%d].profiles and dependencies[%d].profile & dependencies[%d].profileParents cannot be used together", index, index, index)
		}
	}

	return nil
}

func isDependencyNameUnique(name string, dependencies []*latest.DependencyConfig) bool {
	found := false
	for _, d := range dependencies {
		if d.Name == name {
			if found {
				return false
			}

			found = true
		}
	}

	return true
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
		if len(hookConfig.Events) == 0 {
			return errors.Errorf("hooks[%d].events is required", index)
		}
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
		if hookConfig.Upload != nil && hookConfig.Container == nil {
			return errors.Errorf("hooks[%d].container is required if hooks[%d].upload is used", index, index)
		}
		if hookConfig.Download != nil && hookConfig.Container == nil {
			return errors.Errorf("hooks[%d].container is required if hooks[%d].download is used", index, index)
		}
		if hookConfig.Logs != nil && hookConfig.Container == nil {
			return errors.Errorf("hooks[%d].container is required if hooks[%d].logs is used", index, index)
		}
		if hookConfig.Wait != nil && hookConfig.Container == nil {
			return errors.Errorf("hooks[%d].container is required if hooks[%d].wait is used", index, index)
		}
		if hookConfig.Wait != nil && !hookConfig.Wait.Running && hookConfig.Wait.TerminatedWithCode == nil {
			return errors.Errorf("hooks[%d].wait.running or hooks[%d].wait.terminatedWithCode is required if hooks[%d].wait is used", index, index, index)
		}
		if hookConfig.Container != nil {
			if hookConfig.Container.ContainerName != "" && len(hookConfig.Container.LabelSelector) == 0 {
				return errors.Errorf("hooks[%d].container.containerName is defined but hooks[%d].container.labelSelector is not defined", index, index)
			}
			if len(hookConfig.Container.LabelSelector) == 0 && hookConfig.Container.ImageSelector == "" {
				return errors.Errorf("hooks[%d].container.labelSelector and hooks[%d].container.imageSelector are not defined", index, index)
			}
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
		if deployConfig.Helm != nil {
			if deployConfig.Helm.Chart == nil &&
				(deployConfig.Helm.ComponentChart == nil || !*deployConfig.Helm.ComponentChart) {
				return errors.Errorf("deployments[%d].helm.chart or deployments[%d].helm.componentChart is required", index, index)
			}

			if deployConfig.Helm.Chart != nil &&
				((deployConfig.Helm.Chart.Name != "" && deployConfig.Helm.Chart.Git != nil) ||
					(deployConfig.Helm.Chart.Name == "" && deployConfig.Helm.Chart.Git == nil)) {
				return errors.Errorf("deployments[%d].helm.chart.name or deployments[%d].helm.chart.git is required", index, index)
			}
		}

		if deployConfig.Kubectl != nil && deployConfig.Kubectl.Manifests == nil {
			return errors.Errorf("deployments[%d].kubectl.manifests is required", index)
		}
	}

	return nil
}

func ValidateComponentConfig(deployConfig *latest.DeploymentConfig, overwriteValues map[interface{}]interface{}) error {
	if deployConfig.Helm != nil && deployConfig.Helm.ComponentChart != nil && *deployConfig.Helm.ComponentChart {
		bytes, err := yaml.Marshal(overwriteValues)
		if err != nil {
			return errors.Errorf("deployments[%s].helm: Error marshaling overwrite values: %v", deployConfig.Name, err)
		}

		componentValues := &latest.ComponentConfig{}
		err = yaml.UnmarshalStrict(bytes, componentValues)
		if err != nil {
			return errors.Errorf("deployments[%s].helm.componentChart: component values are incorrect: %v", deployConfig.Name, err)
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
		if _, tag, _ := imageselector.GetStrippedDockerImageName(imageConf.Image); tag != "" {
			return errors.Errorf("images.%s.image '%s' can not have tag '%s'", imageConfigName, imageConf.Image, tag)
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

func isReplacePodsUnique(index int, rp *latest.ReplacePod, rps []*latest.ReplacePod) bool {
	for i, r := range rps {
		if i == index {
			continue
		}

		if r.ImageSelector != "" && r.ImageSelector == rp.ImageSelector {
			return false
		} else if len(r.LabelSelector) > 0 && len(rp.LabelSelector) > 0 && strMapEquals(r.LabelSelector, rp.LabelSelector) {
			return false
		}
	}

	return true
}

func strMapEquals(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if w, ok := b[k]; !ok || v != w {
			return false
		}
	}

	return true
}

func validateDev(config *latest.Config) error {
	for index, rp := range config.Dev.ReplacePods {
		if rp.ContainerName != "" && len(rp.LabelSelector) == 0 {
			return errors.Errorf("Error in config: containerName is defined but label selector is nil in replace pods at index %d", index)
		}

		if len(rp.LabelSelector) == 0 && rp.ImageSelector == "" {
			return errors.Errorf("Error in config: image selector and label selector are nil in replace pods at index %d", index)
		}

		definedSelectors := 0
		if rp.ImageSelector != "" {
			definedSelectors++
		}
		if len(rp.LabelSelector) > 0 {
			definedSelectors++
		}
		if definedSelectors > 1 {
			return errors.Errorf("Error in config: image selector and label selector cannot both be defined in replace pods at index %d", index)
		}
		if !isReplacePodsUnique(index, rp, config.Dev.ReplacePods) {
			return errors.Errorf("Error in config: image selector or label selector is not unique in replace pods at index %d", index)
		}
		for j, p := range rp.PersistPaths {
			if p.Path == "" {
				return errors.Errorf("Error in config: dev.replacePods[%d].persistPaths[%d].path is required", index, j)
			}
		}
	}

	if config.Dev.Ports != nil {
		for index, port := range config.Dev.Ports {
			// Validate imageName and label selector
			if len(port.LabelSelector) == 0 && port.ImageSelector == "" {
				return errors.Errorf("Error in config: image selector and label selector are nil in ports config at index %d", index)
			}

			if len(port.PortMappings) == 0 && len(port.PortMappingsReverse) == 0 {
				return errors.Errorf("Error in config: portMappings is empty in port config at index %d", index)
			}
			if !ValidContainerArch(port.Arch) {
				return errors.Errorf("Error in config: ports.arch is not valid '%s' at index %d", port.Arch, index)
			}
		}
	}

	if config.Dev.Sync != nil {
		for index, sync := range config.Dev.Sync {
			// Validate imageName and label selector
			if len(sync.LabelSelector) == 0 && sync.ImageSelector == "" {
				return errors.Errorf("Error in config: image selector and label selector are nil in sync config at index %d", index)
			}

			// Validate initial sync strategy
			if !ValidInitialSyncStrategy(sync.InitialSync) {
				return errors.Errorf("Error in config: sync.initialSync is not valid '%s' at index %d", sync.InitialSync, index)
			}
			if !ValidContainerArch(sync.Arch) {
				return errors.Errorf("Error in config: sync.arch is not valid '%s' at index %d", sync.Arch, index)
			}
			if sync.OnUpload != nil {
				for j, e := range sync.OnUpload.Exec {
					if e.Command == "" {
						return errors.Errorf("Error in config: dev.sync[%d].exec[%d].command is required", index, j)
					}
				}
			}
		}
	}

	if config.Dev.InteractiveImages != nil {
		for index, imageConf := range config.Dev.InteractiveImages {
			if imageConf.Name == "" {
				return errors.Errorf("Error in config: Unnamed interactive image config at index %d", index)
			}
		}
	}

	if config.Dev.Logs != nil {
		for index, selector := range config.Dev.Logs.Selectors {
			if selector.ImageSelector != "" && len(selector.LabelSelector) > 0 {
				return errors.Errorf("Error in config: dev.logs.selectors[%d].imageSelector and dev.logs.selectors[%d].labelSelector cannot be used together", index, index)
			}
		}
	}

	return nil
}
