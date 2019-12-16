package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/build/builder/helper"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	dockerclient "github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/docker/distribution/reference"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"

	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/term"
	"github.com/pkg/errors"

	"github.com/docker/docker/pkg/jsonmessage"
)

// EngineName is the name of the building engine
const EngineName = "docker"

var (
	stdin, stdout, stderr = term.StdStreams()
)

// Builder holds the necessary information to build and push docker images
type Builder struct {
	helper *helper.BuildHelper

	authConfig *types.AuthConfig
	client     dockerclient.Client
	skipPush   bool
}

// NewBuilder creates a new docker Builder instance
func NewBuilder(config *latest.Config, client dockerclient.Client, kubeClient kubectl.Client, imageConfigName string, imageConf *latest.ImageConfig, imageTag string, skipPush, isDev bool) (*Builder, error) {
	return &Builder{
		helper:   helper.NewBuildHelper(config, kubeClient, EngineName, imageConfigName, imageConf, imageTag, isDev),
		client:   client,
		skipPush: skipPush,
	}, nil
}

// Build implements the interface
func (b *Builder) Build(log logpkg.Logger) error {
	return b.helper.Build(b, log)
}

// ShouldRebuild determines if an image has to be rebuilt
func (b *Builder) ShouldRebuild(cache *generated.CacheConfig, ignoreContextPathChanges bool) (bool, error) {
	return b.helper.ShouldRebuild(cache, ignoreContextPathChanges)
}

// BuildImage builds a dockerimage with the docker cli
// contextPath is the absolute path to the context path
// dockerfilePath is the absolute path to the dockerfile WITHIN the contextPath
func (b *Builder) BuildImage(contextPath, dockerfilePath string, entrypoint []string, cmd []string, log logpkg.Logger) error {
	var (
		fullImageName      = b.helper.ImageName + ":" + b.helper.ImageTag
		displayRegistryURL = "hub.docker.com"
	)

	// Display nice registry name
	registryURL, err := registry.GetRegistryFromImageName(b.helper.ImageName)
	if err != nil {
		return err
	}
	if registryURL != "" {
		displayRegistryURL = registryURL
	}

	// We skip pushing when it is the minikube client
	if b.helper.ImageConf == nil || b.helper.ImageConf.Build == nil || b.helper.ImageConf.Build.Docker == nil || b.helper.ImageConf.Build.Docker.PreferMinikube == nil || *b.helper.ImageConf.Build.Docker.PreferMinikube == true {
		if b.helper.KubeClient != nil && b.helper.KubeClient.IsLocalKubernetes() {
			b.skipPush = true
		}
	}

	// Authenticate
	if b.skipPush == false && (b.helper.ImageConf.Build == nil || b.helper.ImageConf.Build.Docker == nil || b.helper.ImageConf.Build.Docker.SkipPush == nil || *b.helper.ImageConf.Build.Docker.SkipPush == false) {
		log.StartWait("Authenticating (" + displayRegistryURL + ")")
		_, err = b.Authenticate()
		log.StopWait()
		if err != nil {
			return errors.Errorf("Error during image registry authentication: %v", err)
		}

		log.Done("Authentication successful (" + displayRegistryURL + ")")
	}

	// Buildoptions
	options := &types.ImageBuildOptions{}
	if b.helper.ImageConf.Build != nil && b.helper.ImageConf.Build.Docker != nil && b.helper.ImageConf.Build.Docker.Options != nil {
		if b.helper.ImageConf.Build.Docker.Options.BuildArgs != nil {
			options.BuildArgs = b.helper.ImageConf.Build.Docker.Options.BuildArgs
		}
		if b.helper.ImageConf.Build.Docker.Options.Target != "" {
			options.Target = b.helper.ImageConf.Build.Docker.Options.Target
		}
		if b.helper.ImageConf.Build.Docker.Options.Network != "" {
			options.NetworkMode = b.helper.ImageConf.Build.Docker.Options.Network
		}
	}

	// Determine output writer
	var writer io.Writer
	if log == logpkg.GetInstance() {
		writer = stdout
	} else {
		writer = log
	}

	ctx := context.Background()
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
	excludes = append(excludes, ".devspace/")

	buildCtx, err := archive.TarWithOptions(contextDir, &archive.TarOptions{
		ExcludePatterns: excludes,
		ChownOpts:       &idtools.Identity{UID: 0, GID: 0},
	})
	if err != nil {
		return err
	}

	// Check if we should overwrite entrypoint
	if len(entrypoint) > 0 || len(cmd) > 0 {
		dockerfilePath, err = helper.CreateTempDockerfile(dockerfilePath, entrypoint, cmd, options.Target)
		if err != nil {
			return err
		}

		// Check if dockerfile is out of context, then we use the docker way to replace the dockerfile
		if dockerfileCtx != nil {
			// We will add it to the build context
			dockerfileCtx, err = os.Open(dockerfilePath)
			if err != nil {
				return errors.Errorf("unable to open Dockerfile: %v", err)
			}

			defer dockerfileCtx.Close()
		} else {
			// We will add it to the build context
			overwriteDockerfileCtx, err := os.Open(dockerfilePath)
			if err != nil {
				return errors.Errorf("unable to open Dockerfile: %v", err)
			}

			buildCtx, err = helper.OverwriteDockerfileInBuildContext(overwriteDockerfileCtx, buildCtx, relDockerfile)
			if err != nil {
				return errors.Errorf("Error overwriting %s: %v", relDockerfile, err)
			}
		}

		defer os.RemoveAll(filepath.Dir(dockerfilePath))
	}

	// replace Dockerfile if it was added from stdin or a file outside the build-context, and there is archive context
	if dockerfileCtx != nil && buildCtx != nil {
		buildCtx, relDockerfile, err = build.AddDockerfileToBuildContext(dockerfileCtx, buildCtx)
		if err != nil {
			return err
		}
	}

	// Setup an upload progress bar
	outStream := command.NewOutStream(writer)
	progressOutput := streamformatter.NewProgressOutput(outStream)
	body := progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")
	buildOptions := types.ImageBuildOptions{
		Tags:        []string{fullImageName},
		Dockerfile:  relDockerfile,
		BuildArgs:   options.BuildArgs,
		Target:      options.Target,
		NetworkMode: options.NetworkMode,
		AuthConfigs: authConfigs,
	}
	if b.helper.ImageConf.Build != nil && b.helper.ImageConf.Build.Docker != nil && b.helper.ImageConf.Build.Docker.UseBuildKit != nil && *b.helper.ImageConf.Build.Docker.UseBuildKit == true {
		err = b.client.ImageBuildCLI(true, body, writer, buildOptions)
		if err != nil {
			return err
		}
	} else {
		response, err := b.client.ImageBuild(ctx, body, buildOptions)
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
	if b.skipPush == false && (b.helper.ImageConf.Build == nil || b.helper.ImageConf.Build.Docker == nil || b.helper.ImageConf.Build.Docker.SkipPush == nil || *b.helper.ImageConf.Build.Docker.SkipPush == false) {
		err = b.PushImage(writer)
		if err != nil {
			return errors.Errorf("Error during image push: %v", err)
		}

		log.Info("Image pushed to registry (" + displayRegistryURL + ")")
	} else {
		log.Infof("Skip image push for %s", b.helper.ImageName)
	}

	return nil
}

// Authenticate authenticates the client with a remote registry
func (b *Builder) Authenticate() (*types.AuthConfig, error) {
	registryURL, err := registry.GetRegistryFromImageName(b.helper.ImageName + ":" + b.helper.ImageTag)
	if err != nil {
		return nil, err
	}

	b.authConfig, err = b.client.Login(registryURL, "", "", true, false, false)
	if err != nil {
		return nil, err
	}

	return b.authConfig, nil
}

// PushImage pushes an image to the specified registry
func (b *Builder) PushImage(writer io.Writer) error {
	ref, err := reference.ParseNormalizedNamed(b.helper.ImageName + ":" + b.helper.ImageTag)
	if err != nil {
		return err
	}

	encodedAuth, err := encodeAuthToBase64(*b.authConfig)
	if err != nil {
		return err
	}

	out, err := b.client.ImagePush(context.Background(), reference.FamiliarString(ref), types.ImagePushOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return err
	}

	outStream := command.NewOutStream(writer)
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
