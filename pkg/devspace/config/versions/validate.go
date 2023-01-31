package versions

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	jsonyaml "github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	k8sv1 "k8s.io/api/core/v1"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/dockerfile"
	"github.com/loft-sh/devspace/pkg/util/encoding"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
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

func Validate(config *latest.Config) error {
	if config.Name == "" {
		return fmt.Errorf("you need to specify a name for your devspace.yaml")
	}
	if encoding.IsUnsafeName(config.Name) {
		return fmt.Errorf("name has to match the following regex: %v", encoding.UnsafeNameRegEx.String())
	}

	err := validateRequire(config)
	if err != nil {
		return err
	}

	err = validateVars(config.Vars)
	if err != nil {
		return err
	}

	err = validatePipelines(config)
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

	err = validateFunctions(config.Functions)
	if err != nil {
		return err
	}

	return nil
}

func isReservedFunctionName(name string) bool {
	switch name {
	case "true", ":", "false", "exit", "set", "shift", "unset",
		"echo", "printf", "break", "continue", "pwd", "cd",
		"wait", "builtin", "trap", "type", "source", ".", "command",
		"dirs", "pushd", "popd", "umask", "alias", "unalias",
		"fg", "bg", "getopts", "eval", "test", "devspace", "[", "exec",
		"return", "read", "shopt":
		return true
	}
	return false
}

func validateFunctions(functions map[string]string) error {
	for name := range functions {
		if isReservedFunctionName(name) {
			return fmt.Errorf("you cannot use '%s' as a function name as its an internally used special function. Please choose another name", name)
		}
	}

	return nil
}

func validateVars(vars map[string]*latest.Variable) error {
	for i, v := range vars {
		if encoding.IsUnsafeUpperName(v.Name) {
			return fmt.Errorf("vars.%s has to match the following regex: %v", i, encoding.UnsafeUpperNameRegEx.String())
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

func validatePipelines(config *latest.Config) error {
	for name := range config.Pipelines {
		if encoding.IsUnsafeName(name) {
			return fmt.Errorf("pipelines.%s has to match the following regex: %v", name, encoding.UnsafeNameRegEx.String())
		}
	}

	return nil
}

func validateDependencies(config *latest.Config) error {
	for name, dep := range config.Dependencies {
		if encoding.IsUnsafeName(name) {
			return fmt.Errorf("dependencies.%s has to match the following regex: %v", name, encoding.UnsafeNameRegEx.String())
		}
		if dep.Source == nil {
			return errors.Errorf("dependencies.%s.source is required", name)
		}
		if dep.Source.Git == "" && dep.Source.Path == "" {
			return errors.Errorf("dependencies.%s.git or dependencies[%s].path is required", name, name)
		}
	}

	return nil
}

func validateCommands(config *latest.Config) error {
	for key, command := range config.Commands {
		if encoding.IsUnsafeCommandName(command.Name) {
			return fmt.Errorf("commands.%s has to match the following regex: %v", command.Name, encoding.UnsafeNameRegEx.String())
		}
		if command.Command == "" {
			return errors.Errorf("commands.%s.command is required", key)
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
		if encoding.IsUnsafeName(deployConfig.Name) {
			return fmt.Errorf("deployments.%s has to match the following regex: %v", index, encoding.UnsafeNameRegEx.String())
		}
		if deployConfig.Helm == nil && deployConfig.Kubectl == nil && deployConfig.Tanka == nil {
			return errors.Errorf("Please specify either helm, kubectl or tanka as deployment type in deployment %s", deployConfig.Name)
		}
		if deployConfig.Kubectl != nil && deployConfig.Kubectl.Manifests == nil && deployConfig.Kubectl.InlineManifest == "" {
			return errors.Errorf("deployments[%s].kubectl.manifests or deployments[%s].kubectl.InlineManifest is required", index, index)
		}
		if deployConfig.Kubectl != nil && deployConfig.Kubectl.Manifests != nil && deployConfig.Kubectl.InlineManifest != "" {
			return errors.Errorf("deployments[%s].kubectl.manifests and deployments[%s].kubectl.inlineManifest cannot be used together", index, index)
		}
		if deployConfig.Kubectl != nil && deployConfig.Helm != nil {
			return errors.Errorf("deployments[%s].kubectl and deployments[%s].helm cannot be used together", index, index)
		}
		if deployConfig.Kubectl != nil && deployConfig.Tanka != nil {
			return errors.Errorf("deployments[%s].kubectl and deployments[%s].tanka cannot be used together", index, index)
		}
		if deployConfig.Helm != nil && deployConfig.Tanka != nil {
			return errors.Errorf("deployments[%s].helm and deployments[%s].tanka cannot be used together", index, index)
		}
		if deployConfig.Kubectl != nil && deployConfig.Kubectl.Patches != nil {
			for patch := range deployConfig.Kubectl.Patches {
				if deployConfig.Kubectl.Patches[patch].Target.Name == "" {
					return errors.Errorf("deployments[%s].kubectl.patches[%d].target.name is required", index, patch)
				}
				if deployConfig.Kubectl.Patches[patch].Operation == "" {
					return errors.Errorf("deployments[%s].kubectl.patches[%d].op is required", index, patch)
				}
				if deployConfig.Kubectl.Patches[patch].Path == "" {
					return errors.Errorf("deployments[%s].kubectl.patches[%d].path is required", index, patch)
				}
				if deployConfig.Kubectl.Patches[patch].Operation != "remove" &&
					deployConfig.Kubectl.Patches[patch].Value == nil {
					return errors.Errorf("deployments[%s].kubectl.patches[%d].value is required", index, patch)
				}
			}
		}
	}

	return nil
}

func ValidateComponentConfig(deployConfig *latest.DeploymentConfig, overwriteValues map[string]interface{}) error {
	if deployConfig.Helm != nil && deployConfig.Helm.Chart == nil {
		b, err := yaml.Marshal(overwriteValues)
		if err != nil {
			return errors.Errorf("deployments[%s].helm: Error marshaling overwrite values: %v", deployConfig.Name, err)
		}

		componentValues := &latest.ComponentConfig{}
		err = yamlutil.UnmarshalStrict(b, componentValues)
		if err != nil {
			return errors.Errorf("deployments[%s].helm.componentChart: component values are incorrect: %v", deployConfig.Name, err)
		}
	}

	return nil
}

func validatePullSecrets(config *latest.Config) error {
	for _, ps := range config.PullSecrets {
		if encoding.IsUnsafeName(ps.Name) {
			return fmt.Errorf("pullSecrets.%s has to match the following regex: %v", ps.Name, encoding.UnsafeNameRegEx.String())
		}
		if ps.Registry == "" {
			return fmt.Errorf("pullSecrets.%s.registry is required", ps.Name)
		}
	}

	return nil
}

func validateImages(config *latest.Config) error {
	// images lists all the image names in order to check for duplicates
	images := map[string]bool{}
	for imageConfigName, imageConf := range config.Images {
		if encoding.IsUnsafeName(imageConfigName) {
			return fmt.Errorf("images.%s has to match the following regex: %v", imageConfigName, encoding.UnsafeNameRegEx.String())
		}
		if imageConf == nil {
			return errors.Errorf("images.%s is empty and should at least contain an image name", imageConfigName)
		}
		if imageConf.Image == "" {
			return errors.Errorf("images.%s.image is required", imageConfigName)
		}
		if _, tag, _ := dockerfile.GetStrippedDockerImageName(imageConf.Image); tag != "" {
			return errors.Errorf("images.%s.image '%s' can not have tag '%s'", imageConfigName, imageConf.Image, tag)
		}
		if imageConf.Custom != nil && imageConf.Custom.Command == "" && len(imageConf.Custom.Commands) == 0 {
			return errors.Errorf("images.%s.build.custom.command or images.%s.build.custom.commands is required", imageConfigName, imageConfigName)
		}
		if images[imageConf.Image] {
			return errors.Errorf("multiple image definitions with the same image name are not allowed")
		}
		if imageConf.RebuildStrategy != "" && imageConf.RebuildStrategy != latest.RebuildStrategyDefault && imageConf.RebuildStrategy != latest.RebuildStrategyAlways && imageConf.RebuildStrategy != latest.RebuildStrategyIgnoreContextChanges {
			return errors.Errorf("images.%s.rebuildStrategy %s is invalid. Please choose one of %v", imageConfigName, string(imageConf.RebuildStrategy), []latest.RebuildStrategy{latest.RebuildStrategyAlways, latest.RebuildStrategyIgnoreContextChanges})
		}
		if imageConf.Kaniko != nil && imageConf.Kaniko.EnvFrom != nil {
			for _, v := range imageConf.Kaniko.EnvFrom {
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
	for devPodName, devPod := range config.Dev {
		devPodName = strings.TrimSpace(devPodName)
		if encoding.IsUnsafeName(devPodName) {
			return fmt.Errorf("dev.%s has to match the following regex: %v", devPodName, encoding.UnsafeNameRegEx.String())
		}
		if len(devPod.LabelSelector) == 0 && devPod.ImageSelector == "" {
			return errors.Errorf("dev.%s: image selector and label selector are nil", devPodName)
		}

		definedSelectors := 0
		if devPod.ImageSelector != "" {
			definedSelectors++
		}
		if len(devPod.LabelSelector) > 0 {
			definedSelectors++
		}
		if definedSelectors > 1 {
			return errors.Errorf("dev.%s: image selector and label selector cannot be used together", devPodName)
		}

		err := validateDevContainer(fmt.Sprintf("dev.%s", devPodName), &devPod.DevContainer, devPod, false)
		if err != nil {
			return err
		}
		if len(devPod.Containers) > 0 {
			for i, c := range devPod.Containers {
				err := validateDevContainer(fmt.Sprintf("dev.%s.containers[%s]", devPodName, i), c, devPod, true)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func validateDevContainer(path string, devContainer *latest.DevContainer, devPod *latest.DevPod, nameRequired bool) error {
	if nameRequired && devContainer.Container == "" {
		return errors.Errorf("%s.container is required", path)
	}

	if !ValidContainerArch(devContainer.Arch) {
		return errors.Errorf("%s.arch is not valid '%s'", path, devContainer.Arch)
	}

	// check if there are values from devContainers that are overwriting values from devPod
	err := validatePodContainerDuplicates(path, devContainer, devPod)
	if err != nil {
		return err
	}

	for index, sync := range devContainer.Sync {
		// Validate initial sync strategy
		if !ValidInitialSyncStrategy(sync.InitialSync) {
			return errors.Errorf("%s.sync[%d].initialSync is not valid '%s'", path, index, sync.InitialSync)
		}
		if sync.OnUpload != nil {
			for j, e := range sync.OnUpload.Exec {
				if e.Command == "" {
					return errors.Errorf("%s.sync[%d].exec[%d].command is required", path, index, j)
				}
			}
		}
		for j, p := range sync.ExcludePaths {
			if p == "" {
				return errors.Errorf("%s.sync[%d].excludePaths[%d] is empty. This can happen if you use !path without quotes like this: '!path'", path, index, j)
			}
		}
		for j, p := range sync.UploadExcludePaths {
			if p == "" {
				return errors.Errorf("%s.sync[%d].uploadExcludePaths[%d] is empty. This can happen if you use !path without quotes like this: '!path'", path, index, j)
			}
		}
		for j, p := range sync.DownloadExcludePaths {
			if p == "" {
				return errors.Errorf("%s.sync[%d].downloadExcludePaths[%d] is empty. This can happen if you use !path without quotes like this: '!path'", path, index, j)
			}
		}
	}
	for index, port := range devContainer.ReversePorts {
		if port.Port == "" {
			return errors.Errorf("%s.reversePorts[%d].port is required", path, index)
		}
	}
	for j, p := range devContainer.PersistPaths {
		if p.Path == "" {
			return errors.Errorf("%s.persistPaths[%d].path is required", path, j)
		}
	}

	return nil
}

func validatePodContainerDuplicates(path string, devContainer *latest.DevContainer, devPod *latest.DevPod) error {
	if devContainer.Container == "" {
		return nil
	}

	// Extract list of fields from DevContainer struct
	fields := reflect.VisibleFields(reflect.TypeOf(struct{ latest.DevContainer }{}))

	// Make possible to retrieve fields from struct programmatically
	devContainerField := reflect.ValueOf(*devContainer)
	devPodField := reflect.ValueOf(*devPod)

	// iterate trough fields, if a devContainer field is set in devPod,
	// then report error about overwriting
	for _, field := range fields {
		// skip the DevContainer field, check then if a field is valid and set
		if field.Name != "DevContainer" && field.Name != "Container" &&
			devPodField.FieldByName(field.Name).IsValid() && !devPodField.FieldByName(field.Name).IsZero() &&
			!reflect.DeepEqual(devPodField.FieldByName(field.Name).Interface(), devContainerField.FieldByName(field.Name).Interface()) {

			// get DevPod path
			fieldName := string(unicode.ToLower(rune(field.Name[0]))) + field.Name[1:]
			pathFields := strings.Split(path, ".")
			pathFields = pathFields[:len(pathFields)-1]
			sourcepath := strings.Join(pathFields, ".")

			return errors.Errorf("%s.%s will be overwritten by %s, please specify %s.%s instead", sourcepath, fieldName, path, path, fieldName)
		}
	}

	return nil
}
