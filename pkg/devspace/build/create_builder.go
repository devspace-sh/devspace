package build

import (
	"context"

	"github.com/loft-sh/devspace/pkg/devspace/build/builder"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/custom"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/docker"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/kaniko"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	dockerclient "github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/pkg/errors"
)

// createBuilder creates a new builder
func (c *controller) createBuilder(imageConfigName string, imageConf *latest.ImageConfig, imageTags []string, options *Options, log log.Logger) (builder.Interface, error) {
	var err error
	var builder builder.Interface

	if imageConf.Build != nil && imageConf.Build.Custom != nil {
		builder = custom.NewBuilder(imageConfigName, imageConf, imageTags)
	} else if imageConf.Build != nil && imageConf.Build.Docker == nil && imageConf.Build.Kaniko != nil {
		dockerClient, err := dockerclient.NewClient(log)
		if err != nil {
			return nil, errors.Errorf("Error creating docker client: %v", err)
		}

		if c.client == nil {
			// Create kubectl client if not specified
			c.client, err = kubectl.NewDefaultClient()
			if err != nil {
				return nil, errors.Errorf("Unable to create new kubectl client: %v", err)
			}
		}

		log.StartWait("Creating kaniko builder")
		defer log.StopWait()
		builder, err = kaniko.NewBuilder(c.config, dockerClient, c.client, imageConfigName, imageConf, imageTags, log)
		if err != nil {
			return nil, errors.Errorf("Error creating kaniko builder: %v", err)
		}
	} else {
		preferMinikube := true
		if imageConf.Build != nil && imageConf.Build.Docker != nil && imageConf.Build.Docker.PreferMinikube != nil {
			preferMinikube = *imageConf.Build.Docker.PreferMinikube
		}

		kubeContext := ""
		if c.client == nil {
			kubeContext, err = kubeconfig.NewLoader().GetCurrentContext()
			if err != nil {
				return nil, errors.Wrap(err, "get current context")
			}
		} else {
			kubeContext = c.client.CurrentContext()
		}

		dockerClient, err := dockerclient.NewClientWithMinikube(kubeContext, preferMinikube, log)
		if err != nil {
			return nil, errors.Errorf("Error creating docker client: %v", err)
		}

		// Check if docker daemon is running
		_, err = dockerClient.Ping(context.Background())
		if err != nil {
			if imageConf.Build != nil && imageConf.Build.Docker != nil && imageConf.Build.Docker.DisableFallback != nil && *imageConf.Build.Docker.DisableFallback {
				return nil, errors.Errorf("Couldn't reach docker daemon: %v. Is the docker daemon running?", err)
			}

			// Fallback to kaniko
			log.Infof("Couldn't find a running docker daemon. Will fallback to kaniko")
			return c.createBuilder(imageConfigName, convertDockerConfigToKanikoConfig(imageConf), imageTags, options, log)
		}

		builder, err = docker.NewBuilder(c.config, dockerClient, c.client, imageConfigName, imageConf, imageTags, options.SkipPush, options.SkipPushOnLocalKubernetes)
		if err != nil {
			return nil, errors.Errorf("Error creating docker builder: %v", err)
		}
	}

	return builder, nil
}

func convertDockerConfigToKanikoConfig(dockerConfig *latest.ImageConfig) *latest.ImageConfig {
	kanikoBuildOptions := &latest.KanikoConfig{
		Cache: ptr.Bool(true),
	}

	if dockerConfig.Build != nil && dockerConfig.Build.Kaniko != nil {
		kanikoBuildOptions = dockerConfig.Build.Kaniko
	} else if dockerConfig.Build != nil && dockerConfig.Build.Docker != nil && dockerConfig.Build.Docker.Options != nil {
		kanikoBuildOptions.Options = dockerConfig.Build.Docker.Options
	}

	kanikoConfig := &latest.ImageConfig{
		Image:                 dockerConfig.Image,
		Tags:                  dockerConfig.Tags,
		Dockerfile:            dockerConfig.Dockerfile,
		Context:               dockerConfig.Context,
		Entrypoint:            dockerConfig.Entrypoint,
		Cmd:                   dockerConfig.Cmd,
		PreferSyncOverRebuild: dockerConfig.PreferSyncOverRebuild,
		RebuildStrategy:       dockerConfig.RebuildStrategy,
		InjectRestartHelper:   dockerConfig.InjectRestartHelper,
		CreatePullSecret:      dockerConfig.CreatePullSecret,
		Build: &latest.BuildConfig{
			Kaniko: kanikoBuildOptions,
		},
	}

	return kanikoConfig
}
