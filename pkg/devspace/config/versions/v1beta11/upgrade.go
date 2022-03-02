package v1beta11

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	"github.com/loft-sh/devspace/pkg/util/encoding"
	"github.com/loft-sh/devspace/pkg/util/log"
	"k8s.io/apimachinery/pkg/labels"
	"strings"
)

// Upgrade upgrades the config
func (c *Config) Upgrade(log log.Logger) (config.Config, error) {
	clonedConfig := &Config{}
	err := util.Convert(c, clonedConfig)
	if err != nil {
		return nil, err
	}
	clonedConfig.Deployments = nil
	clonedConfig.Dev = DevConfig{}
	nextConfig := &next.Config{}
	err = util.Convert(clonedConfig, nextConfig)
	if err != nil {
		return nil, err
	}

	// just guess a name here
	if nextConfig.Name == "" {
		nextConfig.Name = "devspace"
	}

	// use a pretty simple pipeline which was used by devspace before
	deployPipeline := `run_dependencies_pipelines --all
build_images --all`

	// create the deploy pipeline based on concurrent deployments
	concurrentDeployments := []string{}
	sequentialDeployments := []string{}
	for _, d := range c.Deployments {
		if d.Concurrent {
			concurrentDeployments = append(concurrentDeployments, d.Name)
		} else {
			sequentialDeployments = append(sequentialDeployments, d.Name)
		}
	}

	if len(concurrentDeployments) > 0 {
		deployPipeline += "\ncreate_deployments " + strings.Join(concurrentDeployments, " ")
	}
	if len(sequentialDeployments) > 0 {
		deployPipeline += "\ncreate_deployments " + strings.Join(sequentialDeployments, " ") + " --sequential"
	}

	devPipeline := deployPipeline + "\n" + "start_dev --all"
	if c.Dev.Terminal != nil && c.Dev.Terminal.ImageSelector == "" && len(c.Dev.Terminal.LabelSelector) == 0 {
		devPipeline += "\n" + strings.Join(c.Dev.Terminal.Command, " ")
	}
	nextConfig.Pipelines = map[string]*next.Pipeline{
		"dev": {
			Steps: []next.PipelineStep{
				{
					Run: devPipeline,
				},
			},
		},
		"deploy": {
			Steps: []next.PipelineStep{
				{
					Run: deployPipeline,
				},
			},
		},
	}

	for k, v := range nextConfig.Images {
		delete(nextConfig.Images, k)
		nextConfig.Images[encoding.Convert(k)] = v
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
			Disabled:  deployment.Disabled,
		}
		if deployment.Helm != nil {
			nextConfig.Deployments[name].Helm = &next.HelmConfig{
				ComponentChart: deployment.Helm.ComponentChart,
				Values:         deployment.Helm.Values,
				ValuesFiles:    deployment.Helm.ValuesFiles,
				Wait:           deployment.Helm.Wait,
				DisplayOutput:  deployment.Helm.DisplayOutput,
				Timeout:        deployment.Helm.Timeout,
				Force:          deployment.Helm.Force,
				Atomic:         deployment.Helm.Atomic,
				CleanupOnFail:  deployment.Helm.CleanupOnFail,
				Recreate:       deployment.Helm.Recreate,
				DisableHooks:   deployment.Helm.DisableHooks,
				Driver:         deployment.Helm.Driver,
				Path:           deployment.Helm.Path,
				TemplateArgs:   deployment.Helm.TemplateArgs,
				UpgradeArgs:    deployment.Helm.UpgradeArgs,
				FetchArgs:      deployment.Helm.FetchArgs,
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
					nextConfig.Deployments[name].Helm.Chart.Git = &next.GitSource{
						URL:       deployment.Helm.Chart.Git.URL,
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
			if deployment.Helm.ReplaceImageTags == nil || *deployment.Helm.ReplaceImageTags {
				nextConfig.Deployments[name].Helm.ReplaceImageTags = true
			}
		} else if deployment.Kubectl != nil {
			nextConfig.Deployments[name].Kubectl = &next.KubectlConfig{
				Manifests:     deployment.Kubectl.Manifests,
				Kustomize:     deployment.Kubectl.Kustomize,
				KustomizeArgs: deployment.Kubectl.KustomizeArgs,
				CreateArgs:    deployment.Kubectl.CreateArgs,
				ApplyArgs:     deployment.Kubectl.ApplyArgs,
				CmdPath:       deployment.Kubectl.CmdPath,
			}
			if len(deployment.Kubectl.DeleteArgs) > 0 {
				log.Warnf("deployments[*].kubectl.deleteArgs is not supported anymore in v6")
			}
			if deployment.Kubectl.ReplaceImageTags == nil || *deployment.Kubectl.ReplaceImageTags {
				nextConfig.Deployments[name].Kubectl.ReplaceImageTags = true
			}
		}
	}

	for i, d := range c.Dependencies {
		// dev config for dependencies is not working anymore
		if d.Dev != nil {
			if d.Dev.ReplacePods || d.Dev.Sync || d.Dev.Ports {
				log.Errorf("dependencies[*].dev.replacePods,dependencies[*].dev.sync and dependencies[*].dev.ports is not supported anymore in v6")
				log.Errorf("Please use the dev pipeline instead via 'dependencies[*].pipeline: dev' which will start sync and port-forwarding as well")
			}
		}

		// we use dependency name as override name to identify it
		nextConfig.Dependencies[i].OverrideName = encoding.Convert(d.Name)

		// profile parents are removed
		if len(d.ProfileParents) > 0 {
			nextConfig.Dependencies[i].Profiles = append(nextConfig.Dependencies[i].Profiles, d.ProfileParents...)
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
	for _, o := range c.Dev.Open {
		nextConfig.Open = append(nextConfig.Open, &next.OpenConfig{URL: o.URL})
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
				From:      p.From,
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
		devContainer.ReplaceImage = replacePod.ReplaceImage
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
			devPod.Forward = append(devPod.Forward, &next.PortMapping{
				LocalPort:   pr.LocalPort,
				RemotePort:  pr.RemotePort,
				BindAddress: pr.BindAddress,
			})
		}

		if len(portForwarding.PortMappingsReverse) > 0 {
			devContainer := getMatchingDevContainer(devPod, portForwarding.ContainerName)
			devContainer.Arch = next.ContainerArchitecture(portForwarding.Arch)
			for _, pr := range portForwarding.PortMappingsReverse {
				devContainer.PortMappingsReverse = append(devContainer.PortMappingsReverse, &next.PortMapping{
					LocalPort:   pr.LocalPort,
					RemotePort:  pr.RemotePort,
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
			LocalSubPath:         syncConfig.LocalSubPath,
			ContainerPath:        syncConfig.ContainerPath,
			ExcludePaths:         syncConfig.ExcludePaths,
			ExcludeFile:          syncConfig.ExcludeFile,
			DownloadExcludePaths: syncConfig.DownloadExcludePaths,
			DownloadExcludeFile:  syncConfig.DownloadExcludeFile,
			UploadExcludePaths:   syncConfig.UploadExcludePaths,
			UploadExcludeFile:    syncConfig.UploadExcludeFile,
			InitialSync:          next.InitialSyncStrategy(syncConfig.InitialSync),
			InitialSyncCompareBy: next.InitialSyncCompareBy(syncConfig.InitialSyncCompareBy),
			DisableDownload:      syncConfig.DisableDownload,
			DisableUpload:        syncConfig.DisableUpload,
			Polling:              syncConfig.Polling,
			WaitInitialSync:      syncConfig.WaitInitialSync,
			OnUpload:             nil,
			OnDownload:           nil,
		}
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
			nextSyncConfig.OnDownload = &next.SyncOnDownload{}
			if syncConfig.OnDownload.ExecLocal != nil {
				nextSyncConfig.OnDownload.ExecLocal = &next.SyncExecCommand{
					Command: syncConfig.OnDownload.ExecLocal.Command,
					Args:    syncConfig.OnDownload.ExecLocal.Args,
				}
				if syncConfig.OnDownload.ExecLocal.OnBatch != nil {
					nextSyncConfig.OnDownload.ExecLocal.OnBatch = &next.SyncCommand{
						Command: syncConfig.OnDownload.ExecLocal.OnBatch.Command,
						Args:    syncConfig.OnDownload.ExecLocal.OnBatch.Args,
					}
				}
				if syncConfig.OnDownload.ExecLocal.OnDirCreate != nil {
					nextSyncConfig.OnDownload.ExecLocal.OnDirCreate = &next.SyncCommand{
						Command: syncConfig.OnDownload.ExecLocal.OnDirCreate.Command,
						Args:    syncConfig.OnDownload.ExecLocal.OnDirCreate.Args,
					}
				}
				if syncConfig.OnDownload.ExecLocal.OnFileChange != nil {
					nextSyncConfig.OnDownload.ExecLocal.OnFileChange = &next.SyncCommand{
						Command: syncConfig.OnDownload.ExecLocal.OnFileChange.Command,
						Args:    syncConfig.OnDownload.ExecLocal.OnFileChange.Args,
					}
				}
			}
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
				Disabled:       c.Dev.Terminal.Disabled,
				DisableReplace: true,
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
				Disabled: c.Dev.Logs.Disabled != nil && *c.Dev.Logs.Disabled,
			}
		}
		if c.Dev.Logs.ShowLast != nil {
			log.Warnf("dev.logs.showLast is not supported anymore in DevSpace version 6 and has no effect")
		}
	}

	// flatten dev containers
	for k := range devPods {
		if len(devPods[k].Containers) == 1 {
			devPods[k].DevContainer = devPods[k].Containers[0]
			devPods[k].Containers = nil
		}
	}

	return devPods, nil
}

func getMatchingDevContainer(devPod *next.DevPod, containerName string) *next.DevContainer {
	for key, container := range devPod.Containers {
		if container.Container == containerName {
			return &devPod.Containers[key]
		}
	}

	devPod.Containers = append(devPod.Containers, next.DevContainer{
		Container: containerName,
	})
	return &devPod.Containers[len(devPod.Containers)-1]
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
