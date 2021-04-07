package buildkit

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/docker"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/exec"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	dockerpkg "github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"

	"github.com/docker/docker/api/types"
)

// EngineName is the name of the building engine
const EngineName = "buildkit"

// Builder holds the necessary information to build and push docker images
type Builder struct {
	helper *helper.BuildHelper

	authConfig *types.AuthConfig

	skipPush                  bool
	skipPushOnLocalKubernetes bool
}

// NewBuilder creates a new docker Builder instance
func NewBuilder(config *latest.Config, kubeClient kubectl.Client, imageConfigName string, imageConf *latest.ImageConfig, imageTags []string, skipPush, skipPushOnLocalKubernetes bool) (*Builder, error) {
	return &Builder{
		helper:                    helper.NewBuildHelper(config, kubeClient, EngineName, imageConfigName, imageConf, imageTags),
		skipPush:                  skipPush,
		skipPushOnLocalKubernetes: skipPushOnLocalKubernetes,
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
	// build options
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

	// create the context stream
	body, writer, _, buildOptions, err := docker.CreateContextStream(b.helper, contextPath, dockerfilePath, entrypoint, cmd, options, log)
	if err != nil {
		return err
	}

	buildKitConfig := b.helper.ImageConf.Build.BuildKit

	// create the builder
	builder, err := createBuilder(b.helper.KubeClient, buildKitConfig, log)
	if err != nil {
		return err
	}

	// We skip pushing when it is the minikube client
	if b.skipPushOnLocalKubernetes && b.helper.KubeClient != nil && b.helper.KubeClient.IsLocalKubernetes() {
		b.skipPush = true
	}

	// Should we use the minikube docker daemon?
	useMinikubeDocker := false
	if b.helper.KubeClient != nil && b.helper.KubeClient.CurrentContext() == "minikube" && (buildKitConfig.PreferMinikube == nil || *buildKitConfig.PreferMinikube == true) {
		useMinikubeDocker = true
	}

	// Should we build with cli?
	if b.skipPush {
		buildKitConfig.SkipPush = &b.skipPush
	}

	return buildWithCLI(body, writer, b.helper.KubeClient, builder, buildKitConfig, *buildOptions, useMinikubeDocker, log)
}

func buildWithCLI(context io.Reader, writer io.Writer, kubeClient kubectl.Client, builder string, imageConf *latest.BuildKitConfig, options types.ImageBuildOptions, useMinikubeDocker bool, log logpkg.Logger) error {
	environ := os.Environ()

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
		rawConfig, err := kubeClient.ClientConfig().RawConfig()
		if err != nil {
			return errors.Wrap(err, "get raw kube config")
		}
		if !kubeClient.IsInCluster() {
			rawConfig.CurrentContext = kubeClient.CurrentContext()
		}

		bytes, err := clientcmd.Write(rawConfig)
		if err != nil {
			return err
		}

		tempFile, err := ioutil.TempFile("", "")
		if err != nil {
			return err
		}
		defer os.Remove(tempFile.Name())

		_, err = tempFile.Write(bytes)
		if err != nil {
			return errors.Wrap(err, "error writing to file")
		}

		environ = append(environ, "KUBECONFIG="+tempFile.Name())
		args = append(args, "--builder", builder)
	}

	args = append(args, "-")

	log.Infof("Execute BuildKit command with: %s %s", strings.Join(command, " "), strings.Join(args, " "))
	completeArgs := []string{}
	completeArgs = append(completeArgs, command[1:]...)
	completeArgs = append(completeArgs, args...)

	cmd := exec.Command(command[0], completeArgs...)
	cmd.Env = environ
	if useMinikubeDocker {
		minikubeEnv, err := dockerpkg.GetMinikubeEnvironment()
		if err != nil {
			return fmt.Errorf("error retrieving minikube environment with 'minikube docker-env --shell none'. Try setting the option preferMinikube to false: %v", err)
		}
		for k, v := range minikubeEnv {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	cmd.Stdin = context
	cmd.Stdout = writer
	cmd.Stderr = writer

	return cmd.Run()
}

func createBuilder(kubeClient kubectl.Client, imageConf *latest.BuildKitConfig, log logpkg.Logger) (string, error) {
	if imageConf.InCluster == nil {
		return "", nil
	} else if kubeClient == nil {
		return "", fmt.Errorf("cannot build in cluster wth build kit without a correct kubernetes context")
	}

	namespace := kubeClient.Namespace()
	if imageConf.InCluster != nil && imageConf.InCluster.Namespace != "" {
		namespace = imageConf.InCluster.Namespace
	}

	name := "devspace-" + namespace
	if imageConf.InCluster != nil && imageConf.InCluster.Name != "" {
		name = imageConf.InCluster.Name
	}

	// check if we should skip
	if imageConf.InCluster.NoCreate {
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
