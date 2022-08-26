package versions

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/dockerfile"
	"github.com/pkg/errors"
)

func adjustConfig(config *latest.Config) error {
	if config.Vars != nil {
		newObjs := map[string]*latest.Variable{}
		for name, v := range config.Vars {
			if v == nil {
				continue
			}
			v.Name = name
			newObjs[name] = v
		}
		config.Vars = newObjs
	}
	if config.Commands != nil {
		newObjs := map[string]*latest.CommandConfig{}
		for name, command := range config.Commands {
			if command == nil {
				continue
			}
			command.Name = name
			newObjs[name] = command
		}
		config.Commands = newObjs
	}
	if config.PullSecrets != nil {
		newObjs := map[string]*latest.PullSecretConfig{}
		for name, pullSecret := range config.PullSecrets {
			if pullSecret == nil {
				continue
			}
			pullSecret.Name = name
			newObjs[name] = pullSecret
		}
		config.PullSecrets = newObjs
	}
	if config.Dev != nil {
		newObjs := map[string]*latest.DevPod{}
		for name, devPod := range config.Dev {
			if devPod == nil {
				continue
			}
			devPod.Name = name
			for c, v := range devPod.Containers {
				v.Container = c
			}
			newObjs[name] = devPod
		}
		config.Dev = newObjs
	}
	if config.Pipelines != nil {
		newObjs := map[string]*latest.Pipeline{}
		for name, pipeline := range config.Pipelines {
			if pipeline == nil {
				continue
			}
			pipeline.Name = name
			newObjs[name] = pipeline
		}
		config.Pipelines = newObjs
	}
	if config.Dependencies != nil {
		newObjs := map[string]*latest.DependencyConfig{}
		for name, dep := range config.Dependencies {
			if dep == nil {
				continue
			}
			dep.Name = name
			newObjs[name] = dep
		}
		config.Dependencies = newObjs
	}
	if config.Images != nil {
		newObjs := map[string]*latest.Image{}
		for k, v := range config.Images {
			if v != nil {
				v.Name = k
				image, tag, err := dockerfile.GetStrippedDockerImageName(v.Image)
				if err != nil {
					return errors.Errorf("error parsing images.%s.image: '%s': %v", k, v.Image, err)
				}
				if tag != "" {
					v.Image = image
					oldTags := v.Tags
					v.Tags = []string{}
					v.Tags = append(v.Tags, tag)
					v.Tags = append(v.Tags, oldTags...)
				}

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
	return nil
}
