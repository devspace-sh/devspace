package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/distribution/reference"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	dockerclient "github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	command2 "github.com/loft-sh/loft-util/pkg/command"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"

	"github.com/docker/docker/pkg/jsonmessage"
)

// EngineName is the name of the building engine
const EngineName = "docker"

// Builder holds the necessary information to build and push docker images
type Builder struct {
	helper *helper.BuildHelper

	authConfig                *types.AuthConfig
	client                    dockerclient.Client
	skipPush                  bool
	skipPushOnLocalKubernetes bool
}

// NewBuilder creates a new docker Builder instance
func NewBuilder(ctx devspacecontext.Context, client dockerclient.Client, imageConf *latest.Image, imageTags []string, skipPush, skipPushOnLocalKubernetes bool) (*Builder, error) {
	return &Builder{
		helper:                    helper.NewBuildHelper(ctx, EngineName, imageConf, imageTags),
		client:                    client,
		skipPush:                  skipPush,
		skipPushOnLocalKubernetes: skipPushOnLocalKubernetes,
	}, nil
}

// Build implements the interface
func (b *Builder) Build(ctx devspacecontext.Context) error {
	return b.helper.Build(ctx, b)
}

// ShouldRebuild determines if an image has to be rebuilt
func (b *Builder) ShouldRebuild(ctx devspacecontext.Context, forceRebuild bool) (bool, error) {
	// Check if image is present in local registry
	imageCache, _ := ctx.Config().LocalCache().GetImageCache(b.helper.ImageConf.Name)
	imageName := imageCache.ResolveImage() + ":" + imageCache.Tag
	rebuild, err := b.helper.ShouldRebuild(ctx, forceRebuild)

	// Check if image is present in local docker daemon
	if !rebuild && err == nil {
		if b.skipPushOnLocalKubernetes && ctx.KubeClient() != nil && kubectl.IsLocalKubernetes(ctx.KubeClient()) {
			found, err := b.helper.IsImageAvailableLocally(ctx, b.client)
			if !found && err == nil {
				ctx.Log().Infof("Rebuild image %s because it was not found in local docker daemon", imageName)
				return true, nil
			}
		}
	}

	return rebuild, err
}

// BuildImage builds a dockerimage with the docker cli
// contextPath is the absolute path to the context path
// dockerfilePath is the absolute path to the dockerfile WITHIN the contextPath
func (b *Builder) BuildImage(ctx devspacecontext.Context, contextPath, dockerfilePath string, entrypoint []string, cmd []string) error {
	var (
		displayRegistryURL = "hub.docker.com"
	)

	// Display nice registry name
	registryURL, err := pullsecrets.GetRegistryFromImageName(b.helper.ImageName)
	if err != nil {
		return err
	}
	if registryURL != "" {
		displayRegistryURL = registryURL
	}

	// We skip pushing when it is the minikube client
	if b.skipPushOnLocalKubernetes && ctx.KubeClient() != nil && kubectl.IsLocalKubernetes(ctx.KubeClient()) {
		b.skipPush = true
	}

	// Authenticate
	if !b.skipPush && !b.helper.ImageConf.SkipPush {
		if pullsecrets.IsAzureContainerRegistry(registryURL) {
			ctx.Log().Warn("Using an Azure Container Registry(ACR), skipping authentication. You may need to refresh your credentials by running 'az acr login'")
			b.authConfig, err = b.client.GetAuthConfig(ctx.Context(), registryURL, true)
			if err != nil {
				return err
			}
		} else {
			ctx.Log().Info("Authenticating (" + displayRegistryURL + ")...")
			_, err = b.Authenticate(ctx.Context())
			if err != nil {
				return errors.Errorf("Error during image registry authentication: %v", err)
			}

			ctx.Log().Done("Authentication successful (" + displayRegistryURL + ")")
		}
	}

	// create context stream
	body, writer, outStream, buildOptions, err := b.helper.CreateContextStream(contextPath, dockerfilePath, entrypoint, cmd, ctx.Log())
	defer writer.Close()
	if err != nil {
		return err
	}

	// Should we build with cli?
	useBuildKit := false
	useDockerCli := b.helper.ImageConf.Docker != nil && b.helper.ImageConf.Docker.UseCLI
	cliArgs := []string{}
	if b.helper.ImageConf.Docker != nil {
		cliArgs = b.helper.ImageConf.Docker.Args
		if b.helper.ImageConf.Docker.UseBuildKit {
			useBuildKit = true
		}
	}
	if useDockerCli || useBuildKit || len(cliArgs) > 0 {
		err = b.client.ImageBuildCLI(ctx.Context(), ctx.WorkingDir(), ctx.Environ(), useBuildKit, body, writer, cliArgs, *buildOptions, ctx.Log())
		if err != nil {
			return err
		}
	} else {
		// make sure to use the correct proxy configuration
		buildOptions.BuildArgs = b.client.ParseProxyConfig(buildOptions.BuildArgs)

		response, err := b.client.ImageBuild(ctx.Context(), body, *buildOptions)
		if err != nil {
			return err
		}
		defer response.Body.Close()

		err = jsonmessage.DisplayJSONMessagesStream(response.Body, outStream, outStream.FD(), outStream.IsTerminal(), nil)
		if err != nil {
			return err
		}
	}

	// Check if we skip push
	if !b.skipPush && !b.helper.ImageConf.SkipPush {
		for _, tag := range buildOptions.Tags {
			err = b.pushImage(ctx.Context(), writer, tag)
			if err != nil {
				return errors.Errorf("error during image push: %v", err)
			}

			ctx.Log().Info("Image pushed to registry (" + displayRegistryURL + ")")
		}
	} else if ctx.KubeClient() != nil && kubectl.GetKindContext(ctx.KubeClient().CurrentContext()) != "" {
		// Load image if it is a kind-context
		for _, tag := range buildOptions.Tags {
			command := []string{"kind", "load", "docker-image", "--name", kubectl.GetKindContext(ctx.KubeClient().CurrentContext()), tag}
			completeArgs := []string{}
			completeArgs = append(completeArgs, command[1:]...)
			err = command2.Command(ctx.Context(), ctx.WorkingDir(), ctx.Environ(), writer, writer, nil, command[0], completeArgs...)
			if err != nil {
				ctx.Log().Info(errors.Errorf("error during image load to kind cluster: %v", err))
			}
			ctx.Log().Info("Image loaded to kind cluster")
		}
	} else {
		ctx.Log().Infof("Skip image push for %s", b.helper.ImageName)
	}

	return nil
}

// Authenticate authenticates the client with a remote registry
func (b *Builder) Authenticate(ctx context.Context) (*types.AuthConfig, error) {
	registryURL, err := pullsecrets.GetRegistryFromImageName(b.helper.ImageName + ":" + b.helper.ImageTags[0])
	if err != nil {
		return nil, err
	}

	b.authConfig, err = b.client.Login(ctx, registryURL, "", "", true, false, false)
	if err != nil {
		return nil, err
	}

	return b.authConfig, nil
}

// pushImage pushes an image to the specified registry
func (b *Builder) pushImage(ctx context.Context, writer io.Writer, imageName string) error {
	ref, err := reference.ParseNormalizedNamed(imageName)
	if err != nil {
		return err
	}

	encodedAuth, err := encodeAuthToBase64(*b.authConfig)
	if err != nil {
		return err
	}

	out, err := b.client.ImagePush(ctx, reference.FamiliarString(ref), types.ImagePushOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return err
	}

	outStream := streams.NewOut(writer)
	err = jsonmessage.DisplayJSONMessagesStream(out, outStream, outStream.FD(), outStream.IsTerminal(), nil)
	if err != nil {
		return err
	}

	return nil
}

func encodeAuthToBase64(authConfig types.AuthConfig) (string, error) {
	buf, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buf), nil
}
