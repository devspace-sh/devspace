package buildkit

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/restart"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli/streams"

	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	dockerclient "github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"

	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"

	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/term"
	"github.com/pkg/errors"
)

// EngineName is the name of the building engine
const EngineName = "buildkit"

var (
	stdin, stdout, stderr = term.StdStreams()
)

// Builder holds the necessary information to build and push docker images
type Builder struct {
	helper *helper.BuildHelper

	authConfig *types.AuthConfig
	skipPush   bool
}

// NewBuilder creates a new docker Builder instance
func NewBuilder(config *latest.Config, kubeClient kubectl.Client, imageConfigName string, imageConf *latest.ImageConfig, imageTags []string, skipPush bool) (*Builder, error) {
	return &Builder{
		helper:   helper.NewBuildHelper(config, kubeClient, EngineName, imageConfigName, imageConf, imageTags),
		skipPush: skipPush,
	}, nil
}

// Build implements the interface
func (b *Builder) Build(log logpkg.Logger) error {
	return b.helper.Build(b, log)
}

// ShouldRebuild determines if an image has to be rebuilt
func (b *Builder) ShouldRebuild(cache *generated.CacheConfig, forceRebuild, ignoreContextPathChanges bool) (bool, error) {
	return b.helper.ShouldRebuild(cache, forceRebuild, ignoreContextPathChanges)
}

// BuildImage builds a dockerimage with the docker cli
// contextPath is the absolute path to the context path
// dockerfilePath is the absolute path to the dockerfile WITHIN the contextPath
func (b *Builder) BuildImage(contextPath, dockerfilePath string, entrypoint []string, cmd []string, log logpkg.Logger) error {
	// Buildoptions
	options := &types.ImageBuildOptions{}
	if b.helper.ImageConf.Build != nil && b.helper.ImageConf.Build.BuildKit != nil && b.helper.ImageConf.Build.BuildKit.Options != nil {
		if b.helper.ImageConf.Build.BuildKit.Options.BuildArgs != nil {
			options.BuildArgs = b.helper.ImageConf.Build.BuildKit.Options.BuildArgs
		}
		if b.helper.ImageConf.Build.BuildKit.Options.Target != "" {
			options.Target = b.helper.ImageConf.Build.BuildKit.Options.Target
		}
		if b.helper.ImageConf.Build.BuildKit.Options.Network != "" {
			options.NetworkMode = b.helper.ImageConf.Build.BuildKit.Options.Network
		}
	}

	// Determine output writer
	var writer io.Writer
	if log == logpkg.GetInstance() {
		writer = stdout
	} else {
		writer = log
	}

	contextDir, relDockerfile, err := build.GetContextFromLocalDir(contextPath, dockerfilePath)
	if err != nil {
		return err
	}

	var dockerfileCtx *os.File

	// Dockerfile is out of context
	if strings.HasPrefix(relDockerfile, ".."+string(filepath.Separator)) {
		// Dockerfile is outside of build-context; read the Dockerfile and pass it as dockerfileCtx
		dockerfileCtx, err = os.Open(dockerfilePath)
		if err != nil {
			return errors.Errorf("unable to open Dockerfile: %v", err)
		}
		defer dockerfileCtx.Close()
	}

	// And canonicalize dockerfile name to a platform-independent one
	authConfigs, _ := dockerclient.GetAllAuthConfigs()
	relDockerfile = archive.CanonicalTarNameForPath(relDockerfile)
	excludes, err := helper.ReadDockerignore(contextDir, relDockerfile)
	if err != nil {
		return err
	}

	if err := build.ValidateContextDirectory(contextDir, excludes); err != nil {
		return errors.Errorf("Error checking context: '%s'", err)
	}

	buildCtx, err := archive.TarWithOptions(contextDir, &archive.TarOptions{
		ExcludePatterns: excludes,
		ChownOpts:       &idtools.Identity{UID: 0, GID: 0},
	})
	if err != nil {
		return err
	}

	// Check if we should overwrite entrypoint
	if len(entrypoint) > 0 || len(cmd) > 0 || b.helper.ImageConf.InjectRestartHelper || len(b.helper.ImageConf.AppendDockerfileInstructions) > 0 {
		dockerfilePath, err = helper.RewriteDockerfile(dockerfilePath, entrypoint, cmd, b.helper.ImageConf.AppendDockerfileInstructions, options.Target, b.helper.ImageConf.InjectRestartHelper, log)
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

		// inject the build script
		if b.helper.ImageConf.InjectRestartHelper {
			helperScript, err := restart.LoadRestartHelper(b.helper.ImageConf.RestartHelperPath)
			if err != nil {
				return errors.Wrap(err, "load restart helper")
			}

			buildCtx, err = helper.InjectBuildScriptInContext(helperScript, buildCtx)
			if err != nil {
				return errors.Wrap(err, "inject build script into context")
			}
		}
	}

	// replace Dockerfile if it was added from stdin or a file outside the build-context, and there is archive context
	if dockerfileCtx != nil && buildCtx != nil {
		buildCtx, relDockerfile, err = build.AddDockerfileToBuildContext(dockerfileCtx, buildCtx)
		if err != nil {
			return err
		}
	}

	// Which tags to build
	tags := []string{}
	for _, tag := range b.helper.ImageTags {
		tags = append(tags, b.helper.ImageName+":"+tag)
	}

	// create the builder
	builder, err := createBuilder(b.helper.KubeClient, b.helper.ImageConf.Build.BuildKit, log)
	if err != nil {
		return err
	}

	// Setup an upload progress bar
	outStream := streams.NewOut(writer)
	progressOutput := streamformatter.NewProgressOutput(outStream)
	body := progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")
	buildOptions := types.ImageBuildOptions{
		Tags:        tags,
		Dockerfile:  relDockerfile,
		BuildArgs:   options.BuildArgs,
		Target:      options.Target,
		NetworkMode: options.NetworkMode,
		AuthConfigs: authConfigs,
	}

	// Should we build with cli?
	if b.skipPush {
		b.helper.ImageConf.Build.BuildKit.SkipPush = &b.skipPush
	}

	return buildWithCLI(body, writer, builder, b.helper.ImageConf.Build.BuildKit, buildOptions, log)
}

func buildWithCLI(context io.Reader, writer io.Writer, builder string, imageConf *latest.BuildKitConfig, options types.ImageBuildOptions, log logpkg.Logger) error {
	// TODO: kube context

	command := []string{"docker", "buildx"}
	if len(imageConf.Command) > 0 {
		command = imageConf.Command
	}

	args := []string{"build"}
	if options.BuildArgs != nil {
		for k, v := range options.BuildArgs {
			if v == nil {
				continue
			}

			args = append(args, "--build-arg", k+"="+*v)
		}
	}
	if options.NetworkMode != "" {
		args = append(args, "--network", options.NetworkMode)
	}
	for _, tag := range options.Tags {
		args = append(args, "--tag", tag)
	}
	if imageConf.SkipPush == nil || *imageConf.SkipPush != true {
		if len(options.Tags) > 0 {
			args = append(args, "--push")
		}
	}
	if options.Dockerfile != "" {
		args = append(args, "--file", options.Dockerfile)
	}
	if options.Target != "" {
		args = append(args, "--target", options.Target)
	}
	for _, arg := range imageConf.Args {
		args = append(args, arg)
	}
	if builder != "" {
		args = append(args, "--builder", builder)
	}

	args = append(args, "-")

	log.Infof("Execute BuildKit command with: %s %s", strings.Join(command, " "), strings.Join(args, " "))
	completeArgs := []string{}
	completeArgs = append(completeArgs, command[1:]...)
	completeArgs = append(completeArgs, args...)

	cmd := exec.Command(command[0], completeArgs...)
	cmd.Stdin = context
	cmd.Stdout = writer
	cmd.Stderr = writer
	return cmd.Run()
}

func createBuilder(kubeClient kubectl.Client, imageConf *latest.BuildKitConfig, log logpkg.Logger) (string, error) {
	namespace := kubeClient.Namespace()
	if imageConf.InCluster != nil && imageConf.InCluster.Namespace != "" {
		namespace = imageConf.InCluster.Namespace
	}

	name := "devspace-" + namespace
	if imageConf.InCluster != nil && imageConf.InCluster.Name != "" {
		name = imageConf.InCluster.Name
	}

	// check if we should skip
	if imageConf.InCluster == nil || imageConf.InCluster.Enabled == false {
		return "", nil
	} else if imageConf.InCluster.NoCreate {
		return name, nil
	}

	command := []string{"docker", "buildx"}
	if len(imageConf.Command) > 0 {
		command = imageConf.Command
	}

	args := []string{"create", "--driver", "kubernetes", "--driver-opt", "namespace=" + namespace, "--name", name}
	if imageConf.InCluster.Rootless {
		args = append(args, "--driver-opt", "rootless=true")
	}
	if len(imageConf.InCluster.Args) > 0 {
		args = append(args, imageConf.InCluster.Args...)
	}

	log.Infof("Ensure BuildKit builder with: %s %s", strings.Join(command, " "), strings.Join(args, " "))
	completeArgs := []string{}
	completeArgs = append(completeArgs, command[1:]...)
	completeArgs = append(completeArgs, args...)

	// create the builder
	out, err := exec.Command(command[0], completeArgs...).CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "existing instance") {
			return name, nil
		}

		return "", fmt.Errorf("error creating BuildKit builder: %s => %v", string(out), err)
	}

	return name, nil
}
