package build

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/buildkit"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/custom"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/docker"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/kaniko"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	dockerclient "github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/pkg/errors"
)

// createBuilder creates a new builder
func (c *controller) createBuilder(ctx *devspacecontext.Context, imageConfigName string, imageConf *latest.ImageConfig, imageTags []string, options *Options) (builder.Interface, error) {
	var err error
	var bldr builder.Interface

	if imageConf.Build != nil && imageConf.Build.Custom != nil {
		bldr = custom.NewBuilder(imageConfigName, imageConf, imageTags)
	} else if imageConf.Build != nil && imageConf.Build.BuildKit != nil {
		ctx.Log.StartWait("Creating BuildKit builder")
		defer ctx.Log.StopWait()
		bldr, err = buildkit.NewBuilder(ctx, imageConfigName, imageConf, imageTags, options.SkipPush, options.SkipPushOnLocalKubernetes)
		if err != nil {
			return nil, errors.Errorf("Error creating kaniko builder: %v", err)
		}
	} else if imageConf.Build != nil && imageConf.Build.Docker == nil && imageConf.Build.Kaniko != nil {
		dockerClient, err := dockerclient.NewClient(ctx.Log)
		if err != nil {
			return nil, errors.Errorf("Error creating docker client: %v", err)
		}

		if ctx.KubeClient == nil {
			// Create kubectl client if not specified
			kubeClient, err := kubectl.NewDefaultClient()
			if err != nil {
				return nil, errors.Errorf("Unable to create new kubectl client: %v", err)
			}

			ctx = ctx.WithKubeClient(kubeClient)
		}

		ctx.Log.StartWait("Creating kaniko builder")
		defer ctx.Log.StopWait()
		bldr, err = kaniko.NewBuilder(ctx, dockerClient, imageConfigName, imageConf, imageTags)
		if err != nil {
			return nil, errors.Errorf("Error creating kaniko builder: %v", err)
		}
	} else {
		preferMinikube := true
		if imageConf.Build != nil && imageConf.Build.Docker != nil && imageConf.Build.Docker.PreferMinikube != nil {
			preferMinikube = *imageConf.Build.Docker.PreferMinikube
		}

		kubeContext := ""
		if ctx.KubeClient == nil {
			kubeContext, err = kubeconfig.NewLoader().GetCurrentContext()
			if err != nil {
				return nil, errors.Wrap(err, "get current context")
			}
		} else {
			kubeContext = ctx.KubeClient.CurrentContext()
		}

		dockerClient, err := dockerclient.NewClientWithMinikube(kubeContext, preferMinikube, ctx.Log)
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
			ctx.Log.Infof("Couldn't find a running docker daemon. Will fallback to kaniko")
			return c.createBuilder(ctx, imageConfigName, convertDockerConfigToKanikoConfig(imageConf), imageTags, options)
		}

		bldr, err = docker.NewBuilder(ctx, dockerClient, imageConfigName, imageConf, imageTags, options.SkipPush, options.SkipPushOnLocalKubernetes)
		if err != nil {
			return nil, errors.Errorf("Error creating docker builder: %v", err)
		}
	}

	// create image pull secret if possible
	if ctx.KubeClient != nil && (imageConf.CreatePullSecret == nil || *imageConf.CreatePullSecret) {
		registryURL, err := pullsecrets.GetRegistryFromImageName(imageConf.Image)
		if err != nil {
			return nil, err
		}

		dockerClient, err := dockerclient.NewClient(ctx.Log)
		if err == nil {
			err = pullsecrets.NewClient(dockerClient).EnsurePullSecret(ctx, ctx.KubeClient.Namespace(), registryURL)
			if err != nil {
				ctx.Log.Errorf("error ensuring pull secret for registry %s: %v", registryURL, err)
			}
		}
	}

	return bldr, nil
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
		Image:               dockerConfig.Image,
		Tags:                dockerConfig.Tags,
		Dockerfile:          dockerConfig.Dockerfile,
		Context:             dockerConfig.Context,
		Entrypoint:          dockerConfig.Entrypoint,
		Cmd:                 dockerConfig.Cmd,
		RebuildStrategy:     dockerConfig.RebuildStrategy,
		InjectRestartHelper: dockerConfig.InjectRestartHelper,
		CreatePullSecret:    dockerConfig.CreatePullSecret,
		Build: &latest.BuildConfig{
			Kaniko: kanikoBuildOptions,
		},
	}

	return kanikoConfig
}
