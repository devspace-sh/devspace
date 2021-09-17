package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/build/builder/restart"
	"github.com/loft-sh/devspace/pkg/devspace/config"

	"github.com/docker/cli/cli/streams"

	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	dockerclient "github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"

	"github.com/docker/distribution/reference"

	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"

	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	dockerterm "github.com/moby/term"
	"github.com/pkg/errors"

	"github.com/docker/docker/pkg/jsonmessage"
)

// EngineName is the name of the building engine
const EngineName = "docker"

var (
	_, stdout, _ = dockerterm.StdStreams()
)

// Builder holds the necessary information to build and push docker images
type Builder struct {
	helper *helper.BuildHelper

	authConfig                *types.AuthConfig
	client                    dockerclient.Client
	skipPush                  bool
	skipPushOnLocalKubernetes bool
}

// NewBuilder creates a new docker Builder instance
func NewBuilder(config config.Config, client dockerclient.Client, kubeClient kubectl.Client, imageConfigName string, imageConf *latest.ImageConfig, imageTags []string, skipPush, skipPushOnLocalKubernetes bool) (*Builder, error) {
	return &Builder{
		helper:                    helper.NewBuildHelper(config, kubeClient, EngineName, imageConfigName, imageConf, imageTags),
		client:                    client,
		skipPush:                  skipPush,
		skipPushOnLocalKubernetes: skipPushOnLocalKubernetes,
	}, nil
}

// Build implements the interface
func (b *Builder) Build(log logpkg.Logger) error {
	return b.helper.Build(b, log)
}

// ShouldRebuild determines if an image has to be rebuilt
func (b *Builder) ShouldRebuild(cache *generated.CacheConfig, forceRebuild bool) (bool, error) {
	return b.helper.ShouldRebuild(cache, forceRebuild)
}

// BuildImage builds a dockerimage with the docker cli
// contextPath is the absolute path to the context path
// dockerfilePath is the absolute path to the dockerfile WITHIN the contextPath
func (b *Builder) BuildImage(contextPath, dockerfilePath string, entrypoint []string, cmd []string, log logpkg.Logger) error {
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
	if b.skipPushOnLocalKubernetes && b.helper.KubeClient != nil && b.helper.KubeClient.IsLocalKubernetes() {
		b.skipPush = true
	}

	// Authenticate
	if !b.skipPush && (b.helper.ImageConf.Build == nil || b.helper.ImageConf.Build.Docker == nil || !b.helper.ImageConf.Build.Docker.SkipPush) {
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

	// create context stream
	body, writer, outStream, buildOptions, err := CreateContextStream(b.helper, contextPath, dockerfilePath, entrypoint, cmd, options, log)
	if err != nil {
		return err
	}

	// Should we build with cli?
	useBuildKit := false
	useDockerCli := b.helper.ImageConf.Build != nil && b.helper.ImageConf.Build.Docker != nil && b.helper.ImageConf.Build.Docker.UseCLI
	cliArgs := []string{}
	if b.helper.ImageConf.Build != nil && b.helper.ImageConf.Build.Docker != nil {
		cliArgs = b.helper.ImageConf.Build.Docker.Args
		if b.helper.ImageConf.Build.Docker.UseBuildKit {
			useBuildKit = true
		}
	}
	if useDockerCli || useBuildKit || len(cliArgs) > 0 {
		err = b.client.ImageBuildCLI(useBuildKit, body, writer, cliArgs, *buildOptions, log)
		if err != nil {
			return err
		}
	} else {
		// make sure to use the correct proxy configuration
		buildOptions.BuildArgs = b.client.ParseProxyConfig(buildOptions.BuildArgs)

		response, err := b.client.ImageBuild(context.Background(), body, *buildOptions)
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
	if !b.skipPush && (b.helper.ImageConf.Build == nil || b.helper.ImageConf.Build.Docker == nil || !b.helper.ImageConf.Build.Docker.SkipPush) {
		for _, tag := range buildOptions.Tags {
			err = b.pushImage(writer, tag)
			if err != nil {
				return errors.Errorf("error during image push: %v", err)
			}

			log.Info("Image pushed to registry (" + displayRegistryURL + ")")
		}
	} else {
		log.Infof("Skip image push for %s", b.helper.ImageName)
	}

	return nil
}

// Authenticate authenticates the client with a remote registry
func (b *Builder) Authenticate() (*types.AuthConfig, error) {
	registryURL, err := pullsecrets.GetRegistryFromImageName(b.helper.ImageName + ":" + b.helper.ImageTags[0])
	if err != nil {
		return nil, err
	}

	b.authConfig, err = b.client.Login(registryURL, "", "", true, false, false)
	if err != nil {
		return nil, err
	}

	return b.authConfig, nil
}

// pushImage pushes an image to the specified registry
func (b *Builder) pushImage(writer io.Writer, imageName string) error {
	ref, err := reference.ParseNormalizedNamed(imageName)
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

	outStream := streams.NewOut(writer)
	err = jsonmessage.DisplayJSONMessagesStream(out, outStream, outStream.FD(), outStream.IsTerminal(), nil)
	if err != nil {
		return err
	}

	return nil
}

// CreateContextStream creates a new context stream that includes the correct docker context, (modified) dockerfile and inject helper
// if needed.
func CreateContextStream(buildHelper *helper.BuildHelper, contextPath, dockerfilePath string, entrypoint, cmd []string, options *types.ImageBuildOptions, log logpkg.Logger) (io.Reader, io.Writer, *streams.Out, *types.ImageBuildOptions, error) {
	// Determine output writer
	var writer io.Writer
	if log == logpkg.GetInstance() {
		writer = stdout
	} else {
		writer = log
	}

	contextDir, relDockerfile, err := build.GetContextFromLocalDir(contextPath, dockerfilePath)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Dockerfile is out of context
	var dockerfileCtx *os.File
	if strings.HasPrefix(relDockerfile, ".."+string(filepath.Separator)) {
		// Dockerfile is outside of build-context; read the Dockerfile and pass it as dockerfileCtx
		dockerfileCtx, err = os.Open(dockerfilePath)
		if err != nil {
			return nil, nil, nil, nil, errors.Errorf("unable to open Dockerfile: %v", err)
		}
		defer dockerfileCtx.Close()
	}

	// And canonicalize dockerfile name to a platform-independent one
	authConfigs, _ := dockerclient.GetAllAuthConfigs()
	relDockerfile = archive.CanonicalTarNameForPath(relDockerfile)
	excludes, err := helper.ReadDockerignore(contextDir, relDockerfile)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	if err := build.ValidateContextDirectory(contextDir, excludes); err != nil {
		return nil, nil, nil, nil, errors.Errorf("Error checking context: '%s'", err)
	}

	buildCtx, err := archive.TarWithOptions(contextDir, &archive.TarOptions{
		ExcludePatterns: excludes,
		ChownOpts:       &idtools.Identity{UID: 0, GID: 0},
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Check if we should overwrite entrypoint
	if len(entrypoint) > 0 || len(cmd) > 0 || buildHelper.ImageConf.InjectRestartHelper || len(buildHelper.ImageConf.AppendDockerfileInstructions) > 0 {
		dockerfilePath, err = helper.RewriteDockerfile(dockerfilePath, entrypoint, cmd, buildHelper.ImageConf.AppendDockerfileInstructions, options.Target, buildHelper.ImageConf.InjectRestartHelper, log)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		// Check if dockerfile is out of context, then we use the docker way to replace the dockerfile
		if dockerfileCtx != nil {
			// We will add it to the build context
			dockerfileCtx, err = os.Open(dockerfilePath)
			if err != nil {
				return nil, nil, nil, nil, errors.Errorf("unable to open Dockerfile: %v", err)
			}

			defer dockerfileCtx.Close()
		} else {
			// We will add it to the build context
			overwriteDockerfileCtx, err := os.Open(dockerfilePath)
			if err != nil {
				return nil, nil, nil, nil, errors.Errorf("unable to open Dockerfile: %v", err)
			}

			buildCtx, err = helper.OverwriteDockerfileInBuildContext(overwriteDockerfileCtx, buildCtx, relDockerfile)
			if err != nil {
				return nil, nil, nil, nil, errors.Errorf("Error overwriting %s: %v", relDockerfile, err)
			}
		}

		defer os.RemoveAll(filepath.Dir(dockerfilePath))

		// inject the build script
		if buildHelper.ImageConf.InjectRestartHelper {
			helperScript, err := restart.LoadRestartHelper(buildHelper.ImageConf.RestartHelperPath)
			if err != nil {
				return nil, nil, nil, nil, errors.Wrap(err, "load restart helper")
			}

			buildCtx, err = helper.InjectBuildScriptInContext(helperScript, buildCtx)
			if err != nil {
				return nil, nil, nil, nil, errors.Wrap(err, "inject build script into context")
			}
		}
	}

	// replace Dockerfile if it was added from stdin or a file outside the build-context, and there is archive context
	if dockerfileCtx != nil && buildCtx != nil {
		buildCtx, relDockerfile, err = build.AddDockerfileToBuildContext(dockerfileCtx, buildCtx)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	// Which tags to build
	tags := []string{}
	for _, tag := range buildHelper.ImageTags {
		tags = append(tags, buildHelper.ImageName+":"+tag)
	}

	// Setup an upload progress bar
	outStream := streams.NewOut(writer)
	progressOutput := streamformatter.NewProgressOutput(outStream)
	body := progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")
	buildOptions := &types.ImageBuildOptions{
		Tags:        tags,
		Dockerfile:  relDockerfile,
		BuildArgs:   options.BuildArgs,
		Target:      options.Target,
		NetworkMode: options.NetworkMode,
		AuthConfigs: authConfigs,
	}

	return body, writer, outStream, buildOptions, nil
}

func encodeAuthToBase64(authConfig types.AuthConfig) (string, error) {
	buf, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buf), nil
}
