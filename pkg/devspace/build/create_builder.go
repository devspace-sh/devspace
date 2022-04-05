package build

import (
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
	"github.com/pkg/errors"
)

// createBuilder creates a new builder
func (c *controller) createBuilder(ctx devspacecontext.Context, imageConfigName string, imageConf *latest.Image, imageTags []string, options *Options) (builder.Interface, error) {
	var err error
	var bldr builder.Interface

	if imageConf.Custom != nil {
		bldr = custom.NewBuilder(imageConfigName, imageConf, imageTags)
	} else if imageConf.BuildKit != nil {
		bldr, err = buildkit.NewBuilder(ctx, imageConfigName, imageConf, imageTags, options.SkipPush, options.SkipPushOnLocalKubernetes)
		if err != nil {
			return nil, errors.Errorf("Error creating kaniko builder: %v", err)
		}
	} else if imageConf.Docker == nil && imageConf.Kaniko != nil {
		if ctx.KubeClient() == nil {
			// Create kubectl client if not specified
			kubeClient, err := kubectl.NewDefaultClient()
			if err != nil {
				return nil, errors.Errorf("Unable to create new kubectl client: %v", err)
			}

			ctx = ctx.WithKubeClient(kubeClient)
		}

		bldr, err = kaniko.NewBuilder(ctx, imageConfigName, imageConf, imageTags)
		if err != nil {
			return nil, errors.Errorf("Error creating kaniko builder: %v", err)
		}
	} else {
		preferMinikube := true
		if imageConf.Docker != nil && imageConf.Docker.PreferMinikube != nil {
			preferMinikube = *imageConf.Docker.PreferMinikube
		}

		kubeContext := ""
		if ctx.KubeClient() == nil {
			kubeContext, err = kubeconfig.NewLoader().GetCurrentContext()
			if err != nil {
				return nil, errors.Wrap(err, "get current context")
			}
		} else {
			kubeContext = ctx.KubeClient().CurrentContext()
		}

		dockerClient, err := dockerclient.NewClientWithMinikube(ctx.Context(), kubeContext, preferMinikube, ctx.Log())
		if err != nil {
			return nil, errors.Errorf("Error creating docker client: %v", err)
		}

		// Check if docker daemon is running
		_, err = dockerClient.Ping(ctx.Context())
		if err != nil {
			if imageConf.Docker != nil && imageConf.Docker.DisableFallback != nil && *imageConf.Docker.DisableFallback {
				return nil, errors.Errorf("Couldn't reach docker daemon: %v. Is the docker daemon running?", err)
			}

			// Fallback to kaniko
			ctx.Log().Infof("Couldn't find a running docker daemon. Will fallback to kaniko")
			return c.createBuilder(ctx, imageConfigName, convertDockerConfigToKanikoConfig(imageConf), imageTags, options)
		}

		bldr, err = docker.NewBuilder(ctx, dockerClient, imageConfigName, imageConf, imageTags, options.SkipPush, options.SkipPushOnLocalKubernetes)
		if err != nil {
			return nil, errors.Errorf("Error creating docker builder: %v", err)
		}
	}

	// create image pull secret if possible
	if ctx.KubeClient() != nil && (imageConf.CreatePullSecret == nil || *imageConf.CreatePullSecret) {
		registryURL, err := pullsecrets.GetRegistryFromImageName(imageConf.Image)
		if err != nil {
			return nil, err
		}

		dockerClient, err := dockerclient.NewClient(ctx.Context(), ctx.Log())
		if err == nil {
			if imageConf.Kaniko != nil && imageConf.Kaniko.Namespace != "" && ctx.KubeClient().Namespace() != imageConf.Kaniko.Namespace {
				err = pullsecrets.NewClient().EnsurePullSecret(ctx, dockerClient, imageConf.Kaniko.Namespace, registryURL)
				if err != nil {
					ctx.Log().Errorf("error ensuring pull secret for registry %s: %v", registryURL, err)
				}
			}

			err = pullsecrets.NewClient().EnsurePullSecret(ctx, dockerClient, ctx.KubeClient().Namespace(), registryURL)
			if err != nil {
				ctx.Log().Errorf("error ensuring pull secret for registry %s: %v", registryURL, err)
			}
		}
	}

	return bldr, nil
}

func convertDockerConfigToKanikoConfig(dockerConfig *latest.Image) *latest.Image {
	kanikoBuildOptions := &latest.KanikoConfig{
		Cache: true,
	}
	if dockerConfig.Kaniko != nil {
		kanikoBuildOptions = dockerConfig.Kaniko
	}
	kanikoConfig := &latest.Image{
		Image:                        dockerConfig.Image,
		Tags:                         dockerConfig.Tags,
		Dockerfile:                   dockerConfig.Dockerfile,
		Context:                      dockerConfig.Context,
		Entrypoint:                   dockerConfig.Entrypoint,
		Cmd:                          dockerConfig.Cmd,
		RebuildStrategy:              dockerConfig.RebuildStrategy,
		InjectRestartHelper:          dockerConfig.InjectRestartHelper,
		AppendDockerfileInstructions: dockerConfig.AppendDockerfileInstructions,
		CreatePullSecret:             dockerConfig.CreatePullSecret,
		BuildArgs:                    dockerConfig.BuildArgs,
		Network:                      dockerConfig.Network,
		Target:                       dockerConfig.Target,
		Kaniko:                       kanikoBuildOptions,
	}

	return kanikoConfig
}
