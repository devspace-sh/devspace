package localregistry

import (
	"context"
	"net/http"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/build/localregistry"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/pkg/errors"
)

// EngineName is the name of the building engine
const EngineName = "localregistry"

// Builder holds the necessary information to build and push docker images
type Builder struct {
	helper *helper.BuildHelper

	localRegistry             *localregistry.LocalRegistry
	skipPush                  bool
	skipPushOnLocalKubernetes bool
}

// NewBuilder creates a new docker Builder instance
func NewBuilder(ctx devspacecontext.Context, localRegistry *localregistry.LocalRegistry, imageConf *latest.Image, imageTags []string, skipPush, skipPushOnLocalKubernetes bool) (*Builder, error) {
	return &Builder{
		helper:                    helper.NewBuildHelper(ctx, EngineName, imageConf, imageTags),
		localRegistry:             localRegistry,
		skipPush:                  skipPush,
		skipPushOnLocalKubernetes: skipPushOnLocalKubernetes,
	}, nil
}

// Build implements the interface
func (b *Builder) Build(ctx devspacecontext.Context) error {
	return b.helper.Build(ctx, b)
}

// BuildImage implements the interface
func (b *Builder) BuildImage(ctx devspacecontext.Context, contextPath string, dockerfilePath string, entrypoint []string, cmd []string) error {
	// create the context stream
	body, writer, _, buildOptions, err := b.helper.CreateContextStream(contextPath, dockerfilePath, entrypoint, cmd, ctx.Log())
	defer writer.Close()
	if err != nil {
		return errors.Wrap(err, "create context stream")
	}

	// replace image names for builder
	for i, image := range buildOptions.Tags {
		buildOptions.Tags[i], err = b.localRegistry.RewriteImageForBuilder(image)
		if err != nil {
			return errors.Wrap(err, "rewrite image")
		}
	}

	// in case of localbuilds, start the local build
	if b.localRegistry.LocalBuild {
		return LocalBuild(ctx, contextPath, dockerfilePath, entrypoint, cmd, b)
	}

	builderPod, err := b.localRegistry.SelectRegistryPod(ctx)
	if err != nil {
		return errors.Wrap(err, "select builder pod")
	}

	// start the remote build
	return RemoteBuild(ctx, builderPod.Name, builderPod.Namespace, body, writer, buildOptions)

}

// ShouldRebuild determines if an image has to be rebuilt
func (b *Builder) ShouldRebuild(ctx devspacecontext.Context, forceRebuild bool) (bool, error) {
	imageCache, _ := ctx.Config().LocalCache().GetImageCache(b.helper.ImageConf.Name)
	if imageCache.Tag != "" {
		if imageCache.IsLocalRegistryImage() {
			imageName := imageCache.ResolveImage()

			found, err := IsImageAvailableRemotely(ctx.Context(), imageName, b)
			if !found && err == nil {
				ctx.Log().Infof("Rebuild image %s because it was not found in the local registry", imageName)
				return true, nil
			}
		}

		registryPod, err := b.localRegistry.SelectRegistryPod(ctx)
		if err != nil {
			return false, err
		}

		imageName := imageCache.ResolveImage() + ":" + imageCache.Tag
		found, err := localregistry.IsImageAvailableInLocalRegistry(ctx, registryPod, imageName)
		if !found && err == nil {
			ctx.Log().Infof("Rebuild image %s because it was not found in the local registry", imageName)
			return true, nil
		}
	}

	return b.helper.ShouldRebuild(ctx, forceRebuild)
}

// IsImageAvailableRemotely will check if current image needs to be built or not
func IsImageAvailableRemotely(ctx context.Context, imageName string, b *Builder) (bool, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return false, err
	}

	image, err := remote.Image(
		ref,
		remote.WithContext(ctx),
		remote.WithTransport(remote.DefaultTransport),
	)
	if err != nil {
		transportError, ok := err.(*transport.Error)
		if ok && transportError.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}

	return image != nil, nil
}
