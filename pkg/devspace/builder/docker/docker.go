package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	dockerclient "github.com/covexo/devspace/pkg/devspace/docker"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/pkg/term"
	"github.com/docker/docker/registry"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"

	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/pkg/errors"

	"github.com/docker/docker/pkg/jsonmessage"
)

var (
	stdin, stdout, stderr = term.StdStreams()
)

// Builder holds the necessary information to build and push docker images
type Builder struct {
	RegistryURL string
	ImageName   string
	ImageTag    string

	imageURL   string
	authConfig *types.AuthConfig
	client     client.CommonAPIClient
}

// NewBuilder creates a new docker Builder instance
func NewBuilder(client client.CommonAPIClient, registryURL, imageName, imageTag string) (*Builder, error) {
	imageURL := imageName + ":" + imageTag
	if registryURL != "" {
		// Check if it's the official registry or not
		ref, err := reference.ParseNormalizedNamed(registryURL + "/" + imageURL)
		if err != nil {
			return nil, err
		}

		repoInfo, err := registry.ParseRepositoryInfo(ref)
		if err != nil {
			return nil, err
		}

		if repoInfo.Index.Official == false {
			imageURL = registryURL + "/" + imageURL
		}
	}

	return &Builder{
		RegistryURL: registryURL,
		ImageName:   imageName,
		ImageTag:    imageTag,
		imageURL:    imageURL,
		client:      client,
	}, nil
}

// BuildImage builds a dockerimage with the docker cli
// contextPath is the absolute path to the context path
// dockerfilePath is the absolute path to the dockerfile WITHIN the contextPath
func (b *Builder) BuildImage(contextPath, dockerfilePath string, options *types.ImageBuildOptions) error {
	if options == nil {
		options = &types.ImageBuildOptions{}
	}

	ctx := context.Background()
	outStream := command.NewOutStream(stdout)
	contextDir, relDockerfile, err := build.GetContextFromLocalDir(contextPath, dockerfilePath)
	if err != nil {
		return err
	}

	var dockerfileCtx *os.File

	// Dockerfile is out of context
	if err == nil && strings.HasPrefix(relDockerfile, ".."+string(filepath.Separator)) {
		// Dockerfile is outside of build-context; read the Dockerfile and pass it as dockerfileCtx
		dockerfileCtx, err = os.Open(dockerfilePath)
		if err != nil {
			return errors.Errorf("unable to open Dockerfile: %v", err)
		}
		defer dockerfileCtx.Close()
	}

	excludes, err := build.ReadDockerignore(contextDir)
	if err != nil {
		return err
	}

	if err := build.ValidateContextDirectory(contextDir, excludes); err != nil {
		return errors.Errorf("Error checking context: '%s'", err)
	}

	// And canonicalize dockerfile name to a platform-independent one
	authConfigs, _ := dockerclient.GetAllAuthConfigs()

	relDockerfile = archive.CanonicalTarNameForPath(relDockerfile)

	excludes = build.TrimBuildFilesFromExcludes(excludes, relDockerfile, false)
	buildCtx, err := archive.TarWithOptions(contextDir, &archive.TarOptions{
		ExcludePatterns: excludes,
		ChownOpts:       &idtools.Identity{UID: 0, GID: 0},
	})
	if err != nil {
		return err
	}

	// replace Dockerfile if it was added from stdin or a file outside the build-context, and there is archive context
	if dockerfileCtx != nil && buildCtx != nil {
		buildCtx, relDockerfile, err = build.AddDockerfileToBuildContext(dockerfileCtx, buildCtx)
		if err != nil {
			return err
		}
	}

	// Setup an upload progress bar
	progressOutput := streamformatter.NewProgressOutput(outStream)
	body := progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")
	response, err := b.client.ImageBuild(ctx, body, types.ImageBuildOptions{
		Tags:        []string{b.imageURL},
		Dockerfile:  relDockerfile,
		BuildArgs:   options.BuildArgs,
		Target:      options.Target,
		NetworkMode: options.NetworkMode,
		AuthConfigs: authConfigs,
	})
	if err != nil {
		return err
	}
	defer response.Body.Close()

	err = jsonmessage.DisplayJSONMessagesStream(response.Body, outStream, outStream.FD(), outStream.IsTerminal(), nil)
	if err != nil {
		return err
	}

	return nil
}

// Authenticate authenticates the client with a remote registry
func (b *Builder) Authenticate(user, password string, checkCredentialsStore bool) (*types.AuthConfig, error) {
	var err error

	b.authConfig, err = dockerclient.Login(b.client, b.RegistryURL, user, password, checkCredentialsStore, false)
	if err != nil {
		return nil, err
	}

	return b.authConfig, nil
}

// PushImage pushes an image to the specified registry
func (b *Builder) PushImage() error {
	ctx := context.Background()
	ref, err := reference.ParseNormalizedNamed(b.imageURL)
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

	outStream := command.NewOutStream(stdout)
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
