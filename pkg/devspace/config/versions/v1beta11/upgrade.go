package v1beta11

import (
	"fmt"
	"path"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"

	"github.com/loft-sh/devspace/pkg/util/ptr"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	"github.com/loft-sh/devspace/pkg/util/encoding"
	"github.com/loft-sh/devspace/pkg/util/log"
	"k8s.io/apimachinery/pkg/labels"
)

// Upgrade upgrades the config
func (c *Config) Upgrade(log log.Logger) (config.Config, error) {
	clonedConfig := &Config{}
	err := util.Convert(c, clonedConfig)
	if err != nil {
		return nil, err
	}
	for i := range clonedConfig.Profiles {
		clonedConfig.Profiles[i].Replace = nil
		clonedConfig.Profiles[i].Merge = nil
		clonedConfig.Profiles[i].StrategicMerge = nil
	}
	clonedConfig.Deployments = nil
	clonedConfig.Dev = DevConfig{}
	clonedConfig.Dependencies = nil
	clonedConfig.Commands = nil
	clonedConfig.PullSecrets = nil
	clonedConfig.Vars = nil
	clonedConfig.Images = nil
	nextConfig := &next.Config{}
	err = util.Convert(clonedConfig, nextConfig)
	if err != nil {
		return nil, err
	}

	// convert profiles
	for i, p := range c.Profiles {
		if p.Merge != nil {
			nextConfig.Profiles[i].Merge = &next.ProfileConfigStructure{
				Images:          p.Merge.Images,
				Dev:             p.Merge.Dev,
				Hooks:           p.Merge.Hooks,
				OldDeployments:  p.Merge.Deployments,
				OldDependencies: p.Merge.Dependencies,
				OldCommands:     p.Merge.Commands,
				OldPullSecrets:  p.Merge.PullSecrets,
				OldVars:         p.Merge.Vars,
			}
		}
		if p.Replace != nil {
			nextConfig.Profiles[i].Replace = &next.ProfileConfigStructure{
				Images:          p.Replace.Images,
				Dev:             p.Replace.Dev,
				Hooks:           p.Replace.Hooks,
				OldDeployments:  p.Replace.Deployments,
				OldDependencies: p.Replace.Dependencies,
				OldCommands:     p.Replace.Commands,
				OldPullSecrets:  p.Replace.PullSecrets,
				OldVars:         p.Replace.Vars,
			}
		}
		if p.StrategicMerge != nil {
			log.Errorf("profiles[*].strategicMerge is not supported anymore in v6")
		}
	}

	// convert vars
	if len(c.Vars) > 0 {
		nextConfig.Vars = map[string]*next.Variable{}
		for _, variable := range c.Vars {
			v := variable
			nextConfig.Vars[v.Name] = &next.Variable{
				Name:              v.Name,
				Question:          v.Question,
				Options:           v.Options,
				Password:          v.Password,
				ValidationPattern: v.ValidationPattern,
				ValidationMessage: v.ValidationMessage,
				NoCache:           v.NoCache,
				AlwaysResolve:     &v.AlwaysResolve,
				Value:             v.Value,
				Default:           v.Default,
				Source:            next.VariableSource(v.Source),
				Command:           v.Command,
				Args:              v.Args,
			}
			for _, c := range v.Commands {
				nextConfig.Vars[v.Name].Commands = append(nextConfig.Vars[v.Name].Commands, next.VariableCommand{
					Command:         c.Command,
					Args:            c.Args,
					OperatingSystem: c.OperatingSystem,
				})
			}
		}
	}

	// add env file
	if nextConfig.Vars == nil {
		nextConfig.Vars = map[string]*next.Variable{}
	}
	nextConfig.Vars["DEVSPACE_ENV_FILE"] = &next.Variable{
		Value:         ".env",
		AlwaysResolve: ptr.Bool(false),
	}

	deployPipeline := ""
	buildPipeline := ""
	purgePipeline := ""
	nextConfig.Dependencies = map[string]*next.DependencyConfig{}
	for _, dep := range c.Dependencies {
		if dep.Source == nil {
			continue
		}

		name := encoding.Convert(dep.Name)

		// dev config for dependencies is not working anymore
		if dep.Dev != nil {
			if dep.Dev.ReplacePods || dep.Dev.Sync || dep.Dev.Ports {
				log.Errorf("dependencies[*].dev.replacePods,dependencies[*].dev.sync and dependencies[*].dev.ports is not supported anymore in v6")
				log.Errorf("Please use the dev pipeline instead via 'dependencies[*].pipeline: dev' which will start sync and port-forwarding as well")
			}
		}
		nextConfig.Dependencies[name] = &next.DependencyConfig{
			Source: &next.SourceConfig{
				Git:            dep.Source.Git,
				CloneArgs:      dep.Source.CloneArgs,
				DisableShallow: dep.Source.DisableShallow,
				DisablePull:    dep.Source.DisableShallow,
				SubPath:        dep.Source.SubPath,
				Branch:         dep.Source.Branch,
				Tag:            dep.Source.Tag,
				Revision:       dep.Source.Revision,
				Path:           dep.Source.Path,
			},
			Profiles:                 dep.Profiles,
			DisableProfileActivation: dep.DisableProfileActivation,
			OverwriteVars:            dep.OverwriteVars,
			IgnoreDependencies:       dep.IgnoreDependencies,
			Namespace:                dep.Namespace,
			Disabled:                 dep.Disabled,
		}
		if dep.Profile != "" {
			nextConfig.Dependencies[name].Profiles = append(nextConfig.Dependencies[name].Profiles, dep.Profile)
		}
		if dep.SkipBuild {
			log.Warnf("dependencies[*].skipBuild is not supported anymore in v6")
		}
		if dep.Source != nil && dep.Source.ConfigName != "" {
			if dep.Source.Path != "" {
				nextConfig.Dependencies[name].Source.Path = path.Join(dep.Source.Path, dep.Source.ConfigName)
			} else {
				nextConfig.Dependencies[name].Source.SubPath = path.Join(dep.Source.SubPath, dep.Source.ConfigName)
			}
		}
		if len(dep.Vars) > 0 {
			nextConfig.Dependencies[name].Vars = map[string]string{}
			for _, v := range dep.Vars {
				nextConfig.Dependencies[name].Vars[v.Name] = v.Value
			}
		}

		if !dep.Disabled {
			deployPipeline += "run_dependencies " + name + "\n"
			buildPipeline += "run_dependencies " + name + " --pipeline build\n"
			purgePipeline += "run_dependencies " + name + " --pipeline purge\n"
		}
	}

	// pull secrets
	pullSecrets := []string{}
	nextConfig.PullSecrets = map[string]*next.PullSecretConfig{}
	pullSecretsByRegistry := map[string]*next.PullSecretConfig{}
	for idx, pullSecret := range c.PullSecrets {
		pullSecretName := fmt.Sprintf("pull-secret-%d", idx)
		pullSecretConfig := &next.PullSecretConfig{
			Name:            pullSecretName,
			Registry:        pullSecret.Registry,
			Username:        pullSecret.Username,
			Password:        pullSecret.Password,
			Email:           pullSecret.Email,
			Secret:          pullSecret.Secret,
			ServiceAccounts: pullSecret.ServiceAccounts,
		}
		nextConfig.PullSecrets[pullSecretName] = pullSecretConfig
		pullSecretsByRegistry[pullSecret.Registry] = pullSecretConfig

		if pullSecret.Disabled {
			continue
		}

		pullSecrets = append(pullSecrets, pullSecretName)
	}

	// Add pull secrets for images for backwards compatibility
	for k, image := range c.Images {
		registryURL, err := pullsecrets.GetRegistryFromImageName(image.Image)
		if err != nil {
			return nil, err
		}

		if registryURL == "" {
			// Skip
			continue
		}

		if image.CreatePullSecret != nil && !*image.CreatePullSecret {
			// Disabled
			continue
		}

		if pullSecretsByRegistry[registryURL] != nil {
			// Already configured
			continue
		}

		// Create a default pull secret config for images without pull secrets.
		pullSecretName := encoding.Convert(k)
		pullSecretConfig := &next.PullSecretConfig{
			Name:     pullSecretName,
			Registry: registryURL,
		}
		nextConfig.PullSecrets[pullSecretName] = pullSecretConfig
		pullSecretsByRegistry[registryURL] = pullSecretConfig

		pullSecrets = append(pullSecrets, pullSecretName)
	}

	// use a pretty simple pipeline which was used by DevSpace before
	if len(pullSecrets) > 0 {
		deployPipeline += fmt.Sprintf("ensure_pull_secrets %s\n", strings.Join(pullSecrets, " "))
	}
	buildImages := []string{}
	nextConfig.Images = map[string]*next.Image{}
	for k, image := range c.Images {
		imageName := encoding.Convert(k)
		nextConfig.Images[imageName] = &next.Image{
			Image:                        image.Image,
			Tags:                         image.Tags,
			Dockerfile:                   image.Dockerfile,
			Context:                      image.Context,
			Entrypoint:                   image.Entrypoint,
			Cmd:                          image.Cmd,
			CreatePullSecret:             image.CreatePullSecret,
			InjectRestartHelper:          false,
			InjectLegacyRestartHelper:    image.InjectRestartHelper,
			RebuildStrategy:              next.RebuildStrategy(image.RebuildStrategy),
			RestartHelperPath:            image.RestartHelperPath,
			AppendDockerfileInstructions: image.AppendDockerfileInstructions,
		}
		if image.RebuildStrategy == "" {
			nextConfig.Images[imageName].RebuildStrategy = next.RebuildStrategyDefault
		}
		if image.Build != nil && image.Build.Kaniko != nil && image.Build.Kaniko.Options != nil {
			nextConfig.Images[imageName].Network = image.Build.Kaniko.Options.Network
			nextConfig.Images[imageName].BuildArgs = image.Build.Kaniko.Options.BuildArgs
			nextConfig.Images[imageName].Target = image.Build.Kaniko.Options.Target
		}
		if image.Build != nil && image.Build.Docker != nil && image.Build.Docker.Options != nil {
			nextConfig.Images[imageName].Network = image.Build.Docker.Options.Network
			nextConfig.Images[imageName].BuildArgs = image.Build.Docker.Options.BuildArgs
			nextConfig.Images[imageName].Target = image.Build.Docker.Options.Target
		}
		if image.Build != nil && image.Build.BuildKit != nil && image.Build.BuildKit.Options != nil {
			nextConfig.Images[imageName].Network = image.Build.BuildKit.Options.Network
			nextConfig.Images[imageName].BuildArgs = image.Build.BuildKit.Options.BuildArgs
			nextConfig.Images[imageName].Target = image.Build.BuildKit.Options.Target
		}
		if image.Build != nil && image.Build.Docker != nil && image.Build.Docker.SkipPush {
			nextConfig.Images[imageName].SkipPush = true
		}
		if image.Build != nil && image.Build.BuildKit != nil && image.Build.BuildKit.SkipPush {
			nextConfig.Images[imageName].SkipPush = true
		}
		if image.Build != nil && image.Build.Docker != nil {
			nextConfig.Images[imageName].Docker = &next.DockerConfig{
				PreferMinikube:  image.Build.Docker.PreferMinikube,
				DisableFallback: image.Build.Docker.DisableFallback,
				UseBuildKit:     image.Build.Docker.UseBuildKit,
				UseCLI:          image.Build.Docker.UseCLI,
				Args:            image.Build.Docker.Args,
			}
		}
		if image.Build != nil && image.Build.BuildKit != nil {
			nextConfig.Images[imageName].BuildKit = &next.BuildKitConfig{
				PreferMinikube: image.Build.BuildKit.PreferMinikube,
				Args:           image.Build.BuildKit.Args,
				Command:        image.Build.BuildKit.Command,
			}
			if image.Build.BuildKit.InCluster != nil {
				nextConfig.Images[imageName].BuildKit.InCluster = &next.BuildKitInClusterConfig{
					Name:         image.Build.BuildKit.InCluster.Name,
					Namespace:    image.Build.BuildKit.InCluster.Namespace,
					Rootless:     image.Build.BuildKit.InCluster.Rootless,
					Image:        image.Build.BuildKit.InCluster.Image,
					NodeSelector: image.Build.BuildKit.InCluster.NodeSelector,
					NoCreate:     image.Build.BuildKit.InCluster.NoCreate,
					NoRecreate:   image.Build.BuildKit.InCluster.NoRecreate,
					NoLoad:       image.Build.BuildKit.InCluster.NoLoad,
					CreateArgs:   image.Build.BuildKit.InCluster.CreateArgs,
				}
			}
		}
		if image.Build != nil && image.Build.Custom != nil {
			nextConfig.Images[imageName].Custom = &next.CustomConfig{
				Command:  image.Build.Custom.Command,
				OnChange: image.Build.Custom.OnChange,

				// Deprecated
				Args:         image.Build.Custom.Args,
				AppendArgs:   image.Build.Custom.AppendArgs,
				ImageFlag:    image.Build.Custom.ImageFlag,
				ImageTagOnly: image.Build.Custom.ImageTagOnly,
			}
			if !image.Build.Custom.SkipImageArg {
				nextConfig.Images[imageName].Custom.SkipImageArg = ptr.Bool(false)
			}
			for _, c := range image.Build.Custom.Commands {
				nextConfig.Images[imageName].Custom.Commands = append(nextConfig.Images[imageName].Custom.Commands, next.CustomConfigCommand{
					Command:         c.Command,
					OperatingSystem: c.OperatingSystem,
				})
			}
		}
		if image.Build != nil && image.Build.Kaniko != nil {
			nextConfig.Images[imageName].Kaniko = &next.KanikoConfig{
				SnapshotMode:        image.Build.Kaniko.SnapshotMode,
				Image:               image.Build.Kaniko.Image,
				InitImage:           image.Build.Kaniko.InitImage,
				Args:                image.Build.Kaniko.Args,
				Command:             image.Build.Kaniko.Command,
				Namespace:           image.Build.Kaniko.Namespace,
				Insecure:            image.Build.Kaniko.Insecure,
				PullSecret:          image.Build.Kaniko.PullSecret,
				SkipPullSecretMount: image.Build.Kaniko.SkipPullSecretMount,
				NodeSelector:        image.Build.Kaniko.NodeSelector,
				Tolerations:         image.Build.Kaniko.Tolerations,
				ServiceAccount:      image.Build.Kaniko.ServiceAccount,
				Annotations:         image.Build.Kaniko.Annotations,
				Labels:              image.Build.Kaniko.Labels,
				InitEnv:             image.Build.Kaniko.InitEnv,
				Env:                 image.Build.Kaniko.Env,
				EnvFrom:             image.Build.Kaniko.EnvFrom,
			}
			if image.Build.Kaniko.Cache == nil || *image.Build.Kaniko.Cache {
				nextConfig.Images[imageName].Kaniko.Cache = true
			}
			if image.Build.Kaniko.Resources != nil {
				nextConfig.Images[imageName].Kaniko.Resources = &next.PodResources{
					Requests: image.Build.Kaniko.Resources.Requests,
					Limits:   image.Build.Kaniko.Resources.Limits,
				}
			}
			for _, c := range image.Build.Kaniko.AdditionalMounts {
				mount := next.KanikoAdditionalMount{
					ReadOnly:  c.ReadOnly,
					MountPath: c.MountPath,
					SubPath:   c.SubPath,
				}
				if c.Secret != nil {
					mount.Secret = &next.KanikoAdditionalMountSecret{
						Name:        c.Secret.Name,
						DefaultMode: c.Secret.DefaultMode,
					}
					for _, item := range c.Secret.Items {
						mount.Secret.Items = append(mount.Secret.Items, next.KanikoAdditionalMountKeyToPath{
							Key:  item.Key,
							Path: item.Path,
							Mode: item.Mode,
						})
					}
				}
				if c.ConfigMap != nil {
					mount.ConfigMap = &next.KanikoAdditionalMountConfigMap{
						Name:        c.ConfigMap.Name,
						DefaultMode: c.ConfigMap.DefaultMode,
					}
					for _, item := range c.ConfigMap.Items {
						mount.ConfigMap.Items = append(mount.ConfigMap.Items, next.KanikoAdditionalMountKeyToPath{
							Key:  item.Key,
							Path: item.Path,
							Mode: item.Mode,
						})
					}
				}
				nextConfig.Images[imageName].Kaniko.AdditionalMounts = append(nextConfig.Images[imageName].Kaniko.AdditionalMounts, mount)
			}
		}

		if c.Images[k].Build != nil && c.Images[k].Build.Disabled {
			continue
		}
		buildImages = append(buildImages, k)
	}
	if len(buildImages) > 0 {
		buildPipeline += fmt.Sprintf("build_images %s\n", strings.Join(buildImages, " "))
		deployPipeline += fmt.Sprintf("build_images %s\n", strings.Join(buildImages, " "))
	}

	// create the deploy pipeline based on concurrent deployments
	concurrentDeployments := []string{}
	sequentialDeployments := []string{}
	for _, d := range c.Deployments {
		if d.Disabled {
			continue
		}
		if d.Concurrent {
			concurrentDeployments = append(concurrentDeployments, d.Name)
		} else {
			sequentialDeployments = append(sequentialDeployments, d.Name)
		}
	}

	prependPurgePipeline := "stop_dev --all\n"
	if len(concurrentDeployments) > 0 {
		prependPurgePipeline += "purge_deployments " + strings.Join(concurrentDeployments, " ") + "\n"
		deployPipeline += "create_deployments " + strings.Join(concurrentDeployments, " ") + "\n"
	}
	if len(sequentialDeployments) > 0 {
		prependPurgePipeline += "purge_deployments " + strings.Join(sequentialDeployments, " ") + " --sequential" + "\n"
		deployPipeline += "create_deployments " + strings.Join(sequentialDeployments, " ") + " --sequential" + "\n"
	}
	purgePipeline = prependPurgePipeline + "\n" + purgePipeline + "\n"

	devPipeline := deployPipeline + "\n" + "start_dev --all" + "\n"
	if c.Dev.Terminal != nil && c.Dev.Terminal.ImageSelector == "" && len(c.Dev.Terminal.LabelSelector) == 0 && len(c.Dev.Terminal.Command) > 0 {
		for _, c := range c.Dev.Terminal.Command {
			devPipeline += "'" + strings.ReplaceAll(c, "'", "'\"'\"'") + "' "
		}

		devPipeline += "\n"
	}

	nextConfig.Pipelines = map[string]*next.Pipeline{
		"build": {
			Run: strings.TrimSpace(buildPipeline),
		},
		"purge": {
			Run: strings.TrimSpace(purgePipeline),
		},
		"dev": {
			Run: strings.TrimSpace(devPipeline),
		},
		"deploy": {
			Run: strings.TrimSpace(deployPipeline),
		},
	}

	nextConfig.Deployments = map[string]*next.DeploymentConfig{}
	for _, deployment := range c.Deployments {
		if deployment.Name == "" {
			continue
		}

		name := encoding.Convert(deployment.Name)
		nextConfig.Deployments[name] = &next.DeploymentConfig{
			Name:      name,
			Namespace: deployment.Namespace,
		}
		if deployment.Helm != nil {
			nextConfig.Deployments[name].Helm = &next.HelmConfig{
				Values:        deployment.Helm.Values,
				ValuesFiles:   deployment.Helm.ValuesFiles,
				DisplayOutput: deployment.Helm.DisplayOutput,
				TemplateArgs:  deployment.Helm.TemplateArgs,
				UpgradeArgs:   deployment.Helm.UpgradeArgs,
			}
			if len(deployment.Helm.FetchArgs) > 0 {
				log.Warnf("deployments[*].helm.fetchArgs is not supported anymore in DevSpace v6")
			}
			if deployment.Helm.Driver != "" {
				log.Warnf("deployments[*].helm.driver is not supported anymore in DevSpace v6")
			}
			if deployment.Helm.Path != "" {
				log.Warnf("deployments[*].helm.path is not supported anymore in DevSpace v6")
			}
			if deployment.Helm.Wait {
				nextConfig.Deployments[name].Helm.UpgradeArgs = append(nextConfig.Deployments[name].Helm.UpgradeArgs, "--wait")
			}
			if deployment.Helm.Timeout != "" {
				nextConfig.Deployments[name].Helm.UpgradeArgs = append(nextConfig.Deployments[name].Helm.UpgradeArgs, "--timeout", deployment.Helm.Timeout)
			}
			if deployment.Helm.Atomic {
				nextConfig.Deployments[name].Helm.UpgradeArgs = append(nextConfig.Deployments[name].Helm.UpgradeArgs, "--atomic")
			}
			if deployment.Helm.CleanupOnFail {
				nextConfig.Deployments[name].Helm.UpgradeArgs = append(nextConfig.Deployments[name].Helm.UpgradeArgs, "--cleanup-on-fail")
			}
			if deployment.Helm.Force {
				nextConfig.Deployments[name].Helm.UpgradeArgs = append(nextConfig.Deployments[name].Helm.UpgradeArgs, "--force")
			}
			if deployment.Helm.DisableHooks {
				nextConfig.Deployments[name].Helm.UpgradeArgs = append(nextConfig.Deployments[name].Helm.UpgradeArgs, "--no-hooks")
			}
			if deployment.Helm.Chart != nil {
				nextConfig.Deployments[name].Helm.Chart = &next.ChartConfig{
					Name:     deployment.Helm.Chart.Name,
					Version:  deployment.Helm.Chart.Version,
					RepoURL:  deployment.Helm.Chart.RepoURL,
					Username: deployment.Helm.Chart.Username,
					Password: deployment.Helm.Chart.Password,
				}
				if deployment.Helm.Chart.Git != nil {
					nextConfig.Deployments[name].Helm.Chart.Source = &next.SourceConfig{
						Git:       deployment.Helm.Chart.Git.URL,
						CloneArgs: deployment.Helm.Chart.Git.CloneArgs,
						Branch:    deployment.Helm.Chart.Git.Branch,
						Tag:       deployment.Helm.Chart.Git.Tag,
						Revision:  deployment.Helm.Chart.Git.Revision,
						SubPath:   deployment.Helm.Chart.Git.SubPath,
					}
				}
			}
			if len(deployment.Helm.DeleteArgs) > 0 {
				log.Warnf("deployments[*].helm.deleteArgs is not supported anymore in v6")
			}
			nextConfig.Deployments[name].UpdateImageTags = deployment.Helm.ReplaceImageTags
		} else if deployment.Kubectl != nil {
			nextConfig.Deployments[name].Kubectl = &next.KubectlConfig{
				Manifests:         deployment.Kubectl.Manifests,
				Kustomize:         deployment.Kubectl.Kustomize,
				KustomizeArgs:     deployment.Kubectl.KustomizeArgs,
				CreateArgs:        deployment.Kubectl.CreateArgs,
				ApplyArgs:         deployment.Kubectl.ApplyArgs,
				KubectlBinaryPath: deployment.Kubectl.CmdPath,
			}
			if len(deployment.Kubectl.DeleteArgs) > 0 {
				log.Warnf("deployments[*].kubectl.deleteArgs is not supported anymore in v6")
			}
			nextConfig.Deployments[name].UpdateImageTags = deployment.Kubectl.ReplaceImageTags
		}
	}

	// interactive mode was removed
	if c.Dev.InteractiveEnabled || len(c.Dev.InteractiveImages) > 0 {
		log.Errorf("Interactive mode is not supported anymore in DevSpace version 6 and has no effect. Please update your config to use terminal instead")
	}

	// dev.autoReload is now unsupported
	if c.Dev.AutoReload != nil {
		log.Errorf("Auto reload is not supported anymore in DevSpace version 6 and has no effect. Please update your config and use rerunning jobs instead")
	}

	// move dev.open -> open
	if len(nextConfig.Dev) > 0 {
		for _, d := range nextConfig.Dev {
			for _, o := range c.Dev.Open {
				d.Open = append(d.Open, &next.OpenConfig{URL: o.URL})
			}
		}
	}

	// commands
	nextConfig.Commands = map[string]*next.CommandConfig{}
	for _, command := range c.Commands {
		commandName := encoding.ConvertCommands(command.Name)
		nextConfig.Commands[commandName] = &next.CommandConfig{
			Name:        commandName,
			Command:     command.Command,
			Args:        command.Args,
			AppendArgs:  command.AppendArgs,
			Description: command.Description,
		}
	}

	// merge dev config together
	devPods, err := c.mergeDevConfig(log)
	if err != nil {
		return nil, err
	}

	nextConfig.Dev = devPods
	return nextConfig, nil
}

func (c *Config) mergeDevConfig(log log.Logger) (map[string]*next.DevPod, error) {
	devPods := map[string]*next.DevPod{}

	// go over replace pods
	for i, replacePod := range c.Dev.ReplacePods {
		if len(replacePod.LabelSelector) == 0 && replacePod.ImageSelector == "" {
			continue
		}

		name := replacePod.Name
		if name == "" {
			name = fmt.Sprintf("replace:%d", i)
		}

		devPod := getMatchingDevPod(devPods, name, replacePod.LabelSelector, replacePod.ImageSelector)
		devPod.Namespace = replacePod.Namespace
		for _, p := range replacePod.Patches {
			devPod.Patches = append(devPod.Patches, &next.PatchConfig{
				Operation: p.Operation,
				Path:      p.Path,
				Value:     p.Value,
			})
		}
		if replacePod.PersistenceOptions != nil {
			devPod.PersistenceOptions = &next.PersistenceOptions{
				Size:             replacePod.PersistenceOptions.Size,
				StorageClassName: replacePod.PersistenceOptions.StorageClassName,
				AccessModes:      replacePod.PersistenceOptions.AccessModes,
				ReadOnly:         replacePod.PersistenceOptions.ReadOnly,
				Name:             replacePod.PersistenceOptions.Name,
			}
		}

		devContainer := getMatchingDevContainer(devPod, replacePod.ContainerName)
		devContainer.DevImage = replacePod.ReplaceImage
		for _, p := range replacePod.PersistPaths {
			nextPersistentPath := next.PersistentPath{
				Path:         p.Path,
				VolumePath:   p.VolumePath,
				ReadOnly:     p.ReadOnly,
				SkipPopulate: p.SkipPopulate,
			}
			if p.InitContainer != nil && p.InitContainer.Resources != nil {
				nextPersistentPath.InitContainer = &next.PersistentPathInitContainer{
					Resources: &next.PodResources{
						Requests: p.InitContainer.Resources.Requests,
						Limits:   p.InitContainer.Resources.Limits,
					},
				}
			}

			devContainer.PersistPaths = append(devContainer.PersistPaths, nextPersistentPath)
		}
	}

	// go over port forwarding
	for i, portForwarding := range c.Dev.Ports {
		if len(portForwarding.LabelSelector) == 0 && portForwarding.ImageSelector == "" {
			continue
		}

		name := portForwarding.Name
		if name == "" {
			name = fmt.Sprintf("ports:%d", i)
		}

		devPod := getMatchingDevPod(devPods, name, portForwarding.LabelSelector, portForwarding.ImageSelector)
		devPod.Namespace = portForwarding.Namespace
		for _, pr := range portForwarding.PortMappings {
			if pr.LocalPort == nil {
				continue
			}
			mapping := fmt.Sprintf("%d", *pr.LocalPort)
			if pr.RemotePort != nil {
				mapping += fmt.Sprintf(":%d", *pr.RemotePort)
			}

			devPod.Ports = append(devPod.Ports, &next.PortMapping{
				Port:        mapping,
				BindAddress: pr.BindAddress,
			})
		}

		if len(portForwarding.PortMappingsReverse) > 0 {
			devContainer := getMatchingDevContainer(devPod, portForwarding.ContainerName)
			devContainer.Arch = next.ContainerArchitecture(portForwarding.Arch)
			for _, pr := range portForwarding.PortMappingsReverse {
				if pr.LocalPort == nil {
					continue
				}
				mapping := fmt.Sprintf("%d", *pr.LocalPort)
				if pr.RemotePort != nil {
					mapping += fmt.Sprintf(":%d", *pr.RemotePort)
				}

				devContainer.ReversePorts = append(devContainer.ReversePorts, &next.PortMapping{
					Port:        mapping,
					BindAddress: pr.BindAddress,
				})
			}
		}
	}

	// go over sync configuration
	printSyncLogs := c.Dev.Logs != nil && (c.Dev.Logs.Sync == nil || *c.Dev.Logs.Sync)
	for i, syncConfig := range c.Dev.Sync {
		if len(syncConfig.LabelSelector) == 0 && syncConfig.ImageSelector == "" {
			continue
		}

		name := syncConfig.Name
		if name == "" {
			name = fmt.Sprintf("sync:%d", i)
		}

		devPod := getMatchingDevPod(devPods, name, syncConfig.LabelSelector, syncConfig.ImageSelector)
		devPod.Namespace = syncConfig.Namespace

		devContainer := getMatchingDevContainer(devPod, syncConfig.ContainerName)
		nextSyncConfig := &next.SyncConfig{
			PrintLogs:            printSyncLogs,
			ExcludePaths:         syncConfig.ExcludePaths,
			ExcludeFile:          syncConfig.ExcludeFile,
			DownloadExcludePaths: syncConfig.DownloadExcludePaths,
			DownloadExcludeFile:  syncConfig.DownloadExcludeFile,
			UploadExcludePaths:   syncConfig.UploadExcludePaths,
			UploadExcludeFile:    syncConfig.UploadExcludeFile,
			InitialSync:          next.InitialSyncStrategy(syncConfig.InitialSync),
			InitialSyncCompareBy: next.InitialSyncCompareBy(syncConfig.InitialSyncCompareBy),
			Polling:              syncConfig.Polling,
			WaitInitialSync:      syncConfig.WaitInitialSync,
		}
		if syncConfig.DisableDownload != nil {
			nextSyncConfig.DisableDownload = *syncConfig.DisableDownload
		}
		if syncConfig.DisableUpload != nil {
			nextSyncConfig.DisableUpload = *syncConfig.DisableUpload
		}
		syncPath := "."
		if syncConfig.LocalSubPath != "" {
			syncPath = syncConfig.LocalSubPath
		}
		if syncConfig.ContainerPath != "" {
			syncPath += ":" + syncConfig.ContainerPath
		} else {
			syncPath += ":."
		}
		nextSyncConfig.Path = syncPath
		if syncConfig.ThrottleChangeDetection != nil {
			log.Errorf("dev.sync[*].throttleChangeDetection is no longer supported in DevSpace version 6 and has no effect")
		}
		if syncConfig.BandwidthLimits != nil {
			nextSyncConfig.BandwidthLimits = &next.BandwidthLimits{
				Download: syncConfig.BandwidthLimits.Download,
				Upload:   syncConfig.BandwidthLimits.Upload,
			}
		}
		if syncConfig.OnUpload != nil {
			nextSyncConfig.OnUpload = &next.SyncOnUpload{
				RestartContainer: syncConfig.OnUpload.RestartContainer,
			}
			for _, e := range syncConfig.OnUpload.Exec {
				nextSyncConfig.OnUpload.Exec = append(nextSyncConfig.OnUpload.Exec, next.SyncExec{
					Name:        e.Name,
					Command:     e.Command,
					Args:        e.Args,
					FailOnError: e.FailOnError,
					Local:       e.Local,
					OnChange:    e.OnChange,
				})
			}
			if syncConfig.OnUpload.ExecRemote != nil {
				nextSyncConfig.OnUpload.ExecRemote = &next.SyncExecCommand{
					Command: syncConfig.OnUpload.ExecRemote.Command,
					Args:    syncConfig.OnUpload.ExecRemote.Args,
				}
				if syncConfig.OnUpload.ExecRemote.OnBatch != nil {
					nextSyncConfig.OnUpload.ExecRemote.OnBatch = &next.SyncCommand{
						Command: syncConfig.OnUpload.ExecRemote.OnBatch.Command,
						Args:    syncConfig.OnUpload.ExecRemote.OnBatch.Args,
					}
				}
				if syncConfig.OnUpload.ExecRemote.OnDirCreate != nil {
					nextSyncConfig.OnUpload.ExecRemote.OnDirCreate = &next.SyncCommand{
						Command: syncConfig.OnUpload.ExecRemote.OnDirCreate.Command,
						Args:    syncConfig.OnUpload.ExecRemote.OnDirCreate.Args,
					}
				}
				if syncConfig.OnUpload.ExecRemote.OnFileChange != nil {
					nextSyncConfig.OnUpload.ExecRemote.OnFileChange = &next.SyncCommand{
						Command: syncConfig.OnUpload.ExecRemote.OnFileChange.Command,
						Args:    syncConfig.OnUpload.ExecRemote.OnFileChange.Args,
					}
				}
			}
		}
		if syncConfig.OnDownload != nil {
			log.Warnf("dev.sync[*].onDownload is not supported anymore in DevSpace v6, please use dev.sync[*].onUpload.exec instead")
		}

		devContainer.Sync = append(devContainer.Sync, nextSyncConfig)
	}

	// convert terminal
	if c.Dev.Terminal != nil {
		if len(c.Dev.Terminal.LabelSelector) > 0 || c.Dev.Terminal.ImageSelector != "" {
			devPod := getMatchingDevPod(devPods, "terminal", c.Dev.Terminal.LabelSelector, c.Dev.Terminal.ImageSelector)
			devPod.Namespace = c.Dev.Terminal.Namespace

			devContainer := getMatchingDevContainer(devPod, c.Dev.Terminal.ContainerName)
			devContainer.Terminal = &next.Terminal{
				Command:        strings.Join(c.Dev.Terminal.Command, " "),
				WorkDir:        c.Dev.Terminal.WorkDir,
				Enabled:        ptr.Bool(!c.Dev.Terminal.Disabled),
				DisableReplace: true,
				DisableScreen:  true,
			}
		}
	}

	// convert logs
	if c.Dev.Logs != nil {
		for _, selector := range c.Dev.Logs.Selectors {
			devPod := getMatchingDevPod(devPods, "logs", selector.LabelSelector, selector.ImageSelector)
			devPod.Namespace = selector.Namespace

			devContainer := getMatchingDevContainer(devPod, selector.ContainerName)
			devContainer.Logs = &next.Logs{
				Enabled: ptr.Bool(!(c.Dev.Logs.Disabled != nil && *c.Dev.Logs.Disabled)),
			}
			if c.Dev.Logs.ShowLast != nil {
				devContainer.Logs.LastLines = int64(*c.Dev.Logs.ShowLast)
			}
		}
	}

	// disable sync replace
	for k := range devPods {
		for i := range devPods[k].Containers {
			if devPods[k].Containers[i].RestartHelper == nil {
				devPods[k].Containers[i].RestartHelper = &next.RestartHelper{}
			}
			devPods[k].Containers[i].RestartHelper.Inject = ptr.Bool(false)
		}
	}

	// flatten dev containers
	for k := range devPods {
		if len(devPods[k].Containers) == 1 {
			for c := range devPods[k].Containers {
				devPods[k].DevContainer = *devPods[k].Containers[c]
				devPods[k].Containers = nil
				break
			}
		}
	}

	return devPods, nil
}

func getMatchingDevContainer(devPod *next.DevPod, containerName string) *next.DevContainer {
	for key, container := range devPod.Containers {
		if container.Container == containerName || container.Container == "" {
			if container.Container == "" && containerName != "" {
				devContainer := container
				devContainer.Container = containerName
				delete(devPod.Containers, key)
				devPod.Containers[containerName] = devContainer
				return devPod.Containers[containerName]
			}
			return devPod.Containers[key]
		} else if containerName == "" {
			return devPod.Containers[key]
		}
	}

	if devPod.Containers == nil {
		devPod.Containers = map[string]*next.DevContainer{}
	}
	devPod.Containers[containerName] = &next.DevContainer{
		Container: containerName,
	}
	return devPod.Containers[containerName]
}

func getMatchingDevPod(devPods map[string]*next.DevPod, name string, labelSelector map[string]string, imageSelector string) *next.DevPod {
	for _, d := range devPods {
		if imageSelector != "" && d.ImageSelector == imageSelector {
			return d
		}
		if len(labelSelector) > 0 && labels.Set(labelSelector).String() == labels.Set(d.LabelSelector).String() {
			return d
		}
	}

	name = encoding.Convert(name)
	devPods[name] = &next.DevPod{
		ImageSelector: imageSelector,
		LabelSelector: labelSelector,
	}
	return devPods[name]
}
