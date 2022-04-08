package versions

import "github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"

func adjustConfig(config *latest.Config) {
	for name, v := range config.Vars {
		v.Name = name
	}
	for name, command := range config.Commands {
		command.Name = name
	}
	for name, pullSecret := range config.PullSecrets {
		pullSecret.Name = name
	}
	for name, devPod := range config.Dev {
		devPod.Name = name
		for c, v := range devPod.Containers {
			v.Container = c
		}
	}
	for name, pipeline := range config.Pipelines {
		pipeline.Name = name
	}
	for name, dep := range config.Dependencies {
		dep.Name = name
	}
	if config.Images != nil {
		newObjs := map[string]*latest.Image{}
		for k, v := range config.Images {
			if v != nil {
				newObjs[k] = v
			}
		}
		config.Images = newObjs
	}
	if config.Deployments != nil {
		newObjs := map[string]*latest.DeploymentConfig{}
		for k, v := range config.Deployments {
			if v != nil {
				v.Name = k
				newObjs[k] = v
			}
		}
		config.Deployments = newObjs
	}
	if config.Hooks != nil {
		newObjs := []*latest.HookConfig{}
		for _, v := range config.Hooks {
			if v != nil {
				newObjs = append(newObjs, v)
			}
		}
		config.Hooks = newObjs
	}
}
