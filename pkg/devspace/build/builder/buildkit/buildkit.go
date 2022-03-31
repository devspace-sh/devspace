package buildkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	command2 "github.com/loft-sh/devspace/pkg/util/command"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/docker/api/types"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/docker"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	dockerpkg "github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"
)

// EngineName is the name of the building engine
const EngineName = "buildkit"

// Builder holds the necessary information to build and push docker images
type Builder struct {
	helper                    *helper.BuildHelper
	skipPush                  bool
	skipPushOnLocalKubernetes bool
}

// NewBuilder creates a new docker Builder instance
func NewBuilder(ctx *devspacecontext.Context, imageConfigName string, imageConf *latest.Image, imageTags []string, skipPush, skipPushOnLocalKubernetes bool) (*Builder, error) {
	// ensure namespace
	if imageConf.BuildKit != nil && imageConf.BuildKit.InCluster != nil && imageConf.BuildKit.InCluster.Namespace != "" {
		err := kubectl.EnsureNamespace(ctx.Context, ctx.KubeClient, imageConf.BuildKit.InCluster.Namespace, ctx.Log)
		if err != nil {
			return nil, err
		}
	}

	return &Builder{
		helper:                    helper.NewBuildHelper(ctx, EngineName, imageConfigName, imageConf, imageTags),
		skipPush:                  skipPush,
		skipPushOnLocalKubernetes: skipPushOnLocalKubernetes,
	}, nil
}

// Build implements the interface
func (b *Builder) Build(ctx *devspacecontext.Context) error {
	return b.helper.Build(ctx, b)
}

// ShouldRebuild determines if an image has to be rebuilt
func (b *Builder) ShouldRebuild(ctx *devspacecontext.Context, forceRebuild bool) (bool, error) {
	rebuild, err := b.helper.ShouldRebuild(ctx, forceRebuild)

	// Check if image is present in local repository
	if !rebuild && err == nil && b.helper.ImageConf.BuildKit.InCluster == nil {
		if b.skipPushOnLocalKubernetes && ctx.KubeClient != nil && kubectl.IsLocalKubernetes(ctx.KubeClient.CurrentContext()) {
			dockerClient, err := dockerpkg.NewClientWithMinikube(ctx.KubeClient.CurrentContext(), b.helper.ImageConf.BuildKit.PreferMinikube == nil || *b.helper.ImageConf.BuildKit.PreferMinikube, ctx.Log)
			if err != nil {
				return false, err
			}

			found, err := b.helper.IsImageAvailableLocally(ctx, dockerClient)
			if !found && err == nil {
				imageCache, _ := ctx.Config.LocalCache().GetImageCache(b.helper.ImageConfigName)
				ctx.Log.Infof("Rebuild image %s because it was not found in local docker daemon", imageCache.ImageName)
				return true, nil
			}
		}
	}

	return rebuild, err
}

// BuildImage builds a dockerimage with the docker cli
// contextPath is the absolute path to the context path
// dockerfilePath is the absolute path to the dockerfile WITHIN the contextPath
func (b *Builder) BuildImage(ctx *devspacecontext.Context, contextPath, dockerfilePath string, entrypoint []string, cmd []string) error {
	// build options
	options := &types.ImageBuildOptions{}
	if b.helper.ImageConf.BuildArgs != nil {
		options.BuildArgs = b.helper.ImageConf.BuildArgs
	}
	if b.helper.ImageConf.Target != "" {
		options.Target = b.helper.ImageConf.Target
	}
	if b.helper.ImageConf.Network != "" {
		options.NetworkMode = b.helper.ImageConf.Network
	}

	buildKitConfig := b.helper.ImageConf.BuildKit

	// create the builder
	builder, err := ensureBuilder(ctx.Context, ctx.WorkingDir, ctx.KubeClient, buildKitConfig, ctx.Log)
	if err != nil {
		return err
	}

	// create the context stream
	body, writer, _, buildOptions, err := docker.CreateContextStream(b.helper, contextPath, dockerfilePath, entrypoint, cmd, options, ctx.Log)
	defer writer.Close()
	if err != nil {
		return err
	}

	// We skip pushing when it is the minikube client
	if b.skipPushOnLocalKubernetes && ctx.KubeClient != nil && kubectl.IsLocalKubernetes(ctx.KubeClient.CurrentContext()) {
		b.skipPush = true
	}

	// Should we use the minikube docker daemon?
	useMinikubeDocker := false
	if ctx.KubeClient != nil && ctx.KubeClient.CurrentContext() == "minikube" && (buildKitConfig.PreferMinikube == nil || *buildKitConfig.PreferMinikube) {
		useMinikubeDocker = true
	}

	// Should we build with cli?
	skipPush := b.skipPush || b.helper.ImageConf.SkipPush
	return buildWithCLI(ctx.Context, ctx.WorkingDir, body, writer, ctx.KubeClient, builder, buildKitConfig, *buildOptions, useMinikubeDocker, skipPush, ctx.Log)
}

func buildWithCLI(ctx context.Context, dir string, context io.Reader, writer io.Writer, kubeClient kubectl.Client, builder string, imageConf *latest.BuildKitConfig, options types.ImageBuildOptions, useMinikubeDocker, skipPush bool, log logpkg.Logger) error {
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
	if !skipPush {
		if len(options.Tags) > 0 {
			args = append(args, "--push")
		}
	} else if builder != "" {
		if imageConf.InCluster == nil || !imageConf.InCluster.NoLoad {
			args = append(args, "--load")
		}
	}
	if options.Dockerfile != "" {
		args = append(args, "--file", options.Dockerfile)
	}
	if options.Target != "" {
		args = append(args, "--target", options.Target)
	}
	if builder != "" {
		tempFile, err := tempKubeContextFromClient(kubeClient)
		if err != nil {
			return err
		}
		defer os.Remove(tempFile)

		environ = append(environ, "KUBECONFIG="+tempFile)
		args = append(args, "--builder", builder)

		// TODO: find a better solution than this
		// we wait here a little bit, otherwise it might be possible that we get issues during
		// parallel image building, as it seems that docker buildx has problems if the
		// same builder is used at the same time for multiple builds and the BuildKit deployment
		// is created in parallel.
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(3000)+500))
	}
	args = append(args, imageConf.Args...)

	args = append(args, "-")

	log.Infof("Execute BuildKit command with: %s %s", strings.Join(command, " "), strings.Join(args, " "))
	completeArgs := []string{}
	completeArgs = append(completeArgs, command[1:]...)
	completeArgs = append(completeArgs, args...)

	var (
		minikubeEnv map[string]string
		err         error
	)
	if useMinikubeDocker {
		minikubeEnv, err = dockerpkg.GetMinikubeEnvironment()
		if err != nil {
			return fmt.Errorf("error retrieving minikube environment with 'minikube docker-env --shell none'. Try setting the option preferMinikube to false: %v", err)
		}
	}

	return command2.CommandWithEnv(ctx, dir, writer, writer, context, minikubeEnv, command[0], completeArgs...)
}

type NodeGroup struct {
	Name    string
	Driver  string
	Nodes   []Node
	Dynamic bool
}

type Node struct {
	Name       string
	Endpoint   string
	Platforms  []interface{}
	Flags      []string
	ConfigFile string
	DriverOpts map[string]string
}

func ensureBuilder(ctx context.Context, workingDir string, kubeClient kubectl.Client, imageConf *latest.BuildKitConfig, log logpkg.Logger) (string, error) {
	if imageConf.InCluster == nil {
		return "", nil
	} else if kubeClient == nil {
		return "", fmt.Errorf("cannot build in cluster wth build kit without a correct kubernetes context")
	}

	namespace := kubeClient.Namespace()
	if imageConf.InCluster.Namespace != "" {
		namespace = imageConf.InCluster.Namespace
	}

	name := "devspace-" + namespace
	if imageConf.InCluster.Name != "" {
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
	if imageConf.InCluster.Image != "" {
		args = append(args, "--driver-opt", "image="+imageConf.InCluster.Image)
	}
	if imageConf.InCluster.NodeSelector != "" {
		args = append(args, "--driver-opt", "nodeselector="+imageConf.InCluster.NodeSelector)
	}
	if len(imageConf.InCluster.CreateArgs) > 0 {
		args = append(args, imageConf.InCluster.CreateArgs...)
	}

	completeArgs := []string{}
	completeArgs = append(completeArgs, command[1:]...)
	completeArgs = append(completeArgs, args...)

	// check if builder already exists
	builderPath := filepath.Join(getConfigStorePath(), "instances", name)
	_, err := os.Stat(builderPath)
	if err == nil {
		if imageConf.InCluster.NoRecreate {
			return name, nil
		}

		// update the builder if necessary
		b, err := ioutil.ReadFile(builderPath)
		if err != nil {
			log.Warnf("Error reading builder %s: %v", builderPath, err)
			return name, nil
		}

		// parse builder config
		ng := &NodeGroup{}
		err = json.Unmarshal(b, ng)
		if err != nil {
			log.Warnf("Error decoding builder %s: %v", builderPath, err)
			return name, nil
		}

		// check for: correct driver name, driver opts
		if strings.ToLower(ng.Driver) == "kubernetes" && len(ng.Nodes) == 1 {
			node := ng.Nodes[0]

			// check driver options
			namespaceCorrect := node.DriverOpts["namespace"] == namespace
			if node.DriverOpts["rootless"] == "" {
				node.DriverOpts["rootless"] = "false"
			}
			rootlessCorrect := strconv.FormatBool(imageConf.InCluster.Rootless) == node.DriverOpts["rootless"]
			imageCorrect := imageConf.InCluster.Image == node.DriverOpts["image"]
			nodeSelectorCorrect := imageConf.InCluster.NodeSelector == node.DriverOpts["nodeselector"]

			// if builder up to date, exit here
			if namespaceCorrect && rootlessCorrect && imageCorrect && nodeSelectorCorrect {
				return name, nil
			}
		}

		// recreate the builder
		log.Infof("Recreate BuildKit builder because builder options differ")

		// create a temporary kube context
		tempFile, err := tempKubeContextFromClient(kubeClient)
		if err != nil {
			return "", err
		}
		defer os.Remove(tempFile)

		// prepare the command
		rmArgs := []string{}
		rmArgs = append(rmArgs, command[1:]...)
		rmArgs = append(rmArgs, "rm", name)

		// execute the command
		out, err := command2.CombinedOutputWithEnv(ctx, workingDir, map[string]string{
			"KUBECONFIG": tempFile,
		}, command[0], rmArgs...)
		if err != nil {
			log.Warnf("error deleting BuildKit builder: %s => %v", string(out), err)
		}
	}

	// create the builder
	log.Infof("Create BuildKit builder with: %s %s", strings.Join(command, " "), strings.Join(args, " "))

	// This is necessary because docker would otherwise save the used kube config
	// which we don't want because we will override it with our own temp kube config
	// during building.
	out, err := command2.CombinedOutputWithEnv(ctx, workingDir, map[string]string{
		"KUBECONFIG": "",
	}, command[0], completeArgs...)
	if err != nil {
		if !strings.Contains(string(out), "existing instance") {
			return "", fmt.Errorf("error creating BuildKit builder: %s => %v", string(out), err)
		}
	}

	return name, nil
}

// getConfigStorePath will look for correct configuration store path;
// if `$BUILDX_CONFIG` is set - use it, otherwise use parent directory
// of Docker config file (i.e. `${DOCKER_CONFIG}/buildx`)
func getConfigStorePath() string {
	if buildxConfig := os.Getenv("BUILDX_CONFIG"); buildxConfig != "" {
		return buildxConfig
	}

	stderr := &bytes.Buffer{}
	configFile := cliconfig.LoadDefaultConfigFile(stderr)
	buildxConfig := filepath.Join(filepath.Dir(configFile.Filename), "buildx")
	return buildxConfig
}

func tempKubeContextFromClient(kubeClient kubectl.Client) (string, error) {
	rawConfig, err := kubeClient.ClientConfig().RawConfig()
	if err != nil {
		return "", errors.Wrap(err, "get raw kube config")
	}
	if !kubeClient.IsInCluster() {
		rawConfig.CurrentContext = kubeClient.CurrentContext()
	}

	bytes, err := clientcmd.Write(rawConfig)
	if err != nil {
		return "", err
	}

	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}

	_, err = tempFile.Write(bytes)
	if err != nil {
		return "", errors.Wrap(err, "error writing to file")
	}

	return tempFile.Name(), nil
}
