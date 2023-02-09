package build

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/buildkit"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/custom"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/docker"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/kaniko"
	localregistry2 "github.com/loft-sh/devspace/pkg/devspace/build/builder/localregistry"
	"github.com/loft-sh/devspace/pkg/devspace/build/localregistry"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	dockerclient "github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/pkg/errors"
)

// createBuilder creates a new builder
func (c *controller) createBuilder(ctx devspacecontext.Context, imageConf *latest.Image, imageTags []string, options *Options) (builder.Interface, error) {
	var err error
	var bldr builder.Interface

	// check if we should use local registry
	if localregistry.UseLocalRegistry(ctx.KubeClient(), ctx.Config().Config(), imageConf, options.SkipPush) && !localregistry.HasPushPermission(imageConf) {
		return localRegistryBuilder(ctx, imageConf, imageTags, options)
	} else {
		// Update cache for non local registry use by default
		imageCache, _ := ctx.Config().LocalCache().GetImageCache(imageConf.Name)
		imageCache.ImageName = imageConf.Image
		imageCache.LocalRegistryImageName = ""
		ctx.Config().LocalCache().SetImageCache(imageConf.Name, imageCache)
	}

	if imageConf.Custom != nil {
		bldr = custom.NewBuilder(imageConf, imageTags)
	} else if imageConf.BuildKit != nil {
		bldr, err = buildkit.NewBuilder(ctx, imageConf, imageTags, options.SkipPush, options.SkipPushOnLocalKubernetes)
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

		bldr, err = kaniko.NewBuilder(ctx, imageConf, imageTags)
		if err != nil {
			return nil, errors.Errorf("Error creating kaniko builder: %v", err)
		}
	} else {
		preferMinikube := true
		if imageConf.Docker != nil && imageConf.Docker.PreferMinikube != nil {
			preferMinikube = *imageConf.Docker.PreferMinikube
		}

		dockerClient, err := dockerclient.NewClientWithMinikube(ctx.Context(), ctx.KubeClient(), preferMinikube, ctx.Log())
		if err != nil {
			return nil, errors.Errorf("Error creating docker client: %v", err)
		}

		// Check if docker daemon is running
		_, err = dockerClient.Ping(ctx.Context())
		if err != nil {
			if imageConf.Docker != nil && imageConf.Docker.DisableFallback != nil && *imageConf.Docker.DisableFallback {
				return nil, errors.Errorf("Couldn't reach docker daemon: %v. Is the docker daemon running?", err)
			}

			// Fallback to local registry
			ctx.Log().Infof("Couldn't find a running docker daemon. Will fallback to local registry")
			return localRegistryBuilder(ctx, imageConf, imageTags, options)
		}

		bldr, err = docker.NewBuilder(ctx, dockerClient, imageConf, imageTags, options.SkipPush, options.SkipPushOnLocalKubernetes)
		if err != nil {
			return nil, errors.Errorf("Error creating docker builder: %v", err)
		}
	}

	return bldr, nil
}

func localRegistryBuilder(ctx devspacecontext.Context, imageConf *latest.Image, imageTags []string, options *Options) (builder.Interface, error) {
	// Not able to deploy a local registry without a valid kube context
	if ctx.KubeClient() == nil {
		return nil, fmt.Errorf("unable to push image %s and a valid kube context is not available", imageConf.Image)
	}

	registryOptions := localregistry.NewDefaultOptions().
		WithNamespace(ctx.KubeClient().Namespace()).
		WithLocalRegistryConfig(ctx.Config().Config().LocalRegistry)

	// Create and start a local registry if one isn't already running
	localRegistry, err := localregistry.GetOrCreateLocalRegistry(ctx, registryOptions)
	if err != nil {
		return nil, errors.Wrap(err, "get or create local registry")
	}

	// Update cache for local registry use
	imageCache, _ := ctx.Config().LocalCache().GetImageCache(imageConf.Name)
	imageCache.ImageName = imageConf.Image
	imageCache.LocalRegistryImageName, err = localRegistry.RewriteImage(imageConf.Image)
	if err != nil {
		return nil, errors.Wrap(err, "rewrite image")
	}
	ctx.Config().LocalCache().SetImageCache(imageConf.Name, imageCache)

	// Create a local registry builder
	bldr, err := localregistry2.NewBuilder(ctx, localRegistry, imageConf, imageTags, options.SkipPush, options.SkipPushOnLocalKubernetes)
	if err != nil {
		return nil, errors.Wrap(err, "create local registry builder")
	}

	return bldr, nil
}
