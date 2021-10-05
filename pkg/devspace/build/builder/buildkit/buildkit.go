package buildkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/docker/api/types"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/docker"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
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
func NewBuilder(config config.Config, kubeClient kubectl.Client, imageConfigName string, imageConf *latest.ImageConfig, imageTags []string, skipPush, skipPushOnLocalKubernetes bool) (*Builder, error) {
	return &Builder{
		helper:                    helper.NewBuildHelper(config, kubeClient, EngineName, imageConfigName, imageConf, imageTags),
		skipPush:                  skipPush,
		skipPushOnLocalKubernetes: skipPushOnLocalKubernetes,
	}, nil
}

// Build implements the interface
func (b *Builder) Build(devspaceID string, log logpkg.Logger) error {
	return b.helper.Build(b, devspaceID, log)
}

// ShouldRebuild determines if an image has to be rebuilt
func (b *Builder) ShouldRebuild(cache *generated.CacheConfig, forceRebuild bool) (bool, error) {
	return b.helper.ShouldRebuild(cache, forceRebuild)
}

// BuildImage builds a dockerimage with the docker cli
// contextPath is the absolute path to the context path
// dockerfilePath is the absolute path to the dockerfile WITHIN the contextPath
func (b *Builder) BuildImage(contextPath, dockerfilePath string, entrypoint []string, cmd []string, devspaceID string, log logpkg.Logger) error {
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

	buildKitConfig := b.helper.ImageConf.Build.BuildKit

	// create the builder
	builder, err := ensureBuilder(b.helper.KubeClient, buildKitConfig, log)
	if err != nil {
		return err
	}

	// create the context stream
	body, writer, _, buildOptions, err := docker.CreateContextStream(b.helper, contextPath, dockerfilePath, entrypoint, cmd, options, log)
	if err != nil {
		return err
	}

	// We skip pushing when it is the minikube client
	if b.skipPushOnLocalKubernetes && b.helper.KubeClient != nil && b.helper.KubeClient.IsLocalKubernetes() {
		b.skipPush = true
	}

	// Should we use the minikube docker daemon?
	useMinikubeDocker := false
	if b.helper.KubeClient != nil && b.helper.KubeClient.CurrentContext() == "minikube" && (buildKitConfig.PreferMinikube == nil || *buildKitConfig.PreferMinikube) {
		useMinikubeDocker = true
	}

	// Should we build with cli?
	if b.skipPush {
		buildKitConfig.SkipPush = b.skipPush
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
	if !imageConf.SkipPush {
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

func ensureBuilder(kubeClient kubectl.Client, imageConf *latest.BuildKitConfig, log logpkg.Logger) (string, error) {
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
		cmd := exec.Command(command[0], rmArgs...)
		cmd.Env = append(os.Environ(), "KUBECONFIG="+tempFile)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Warnf("error deleting BuildKit builder: %s => %v", string(out), err)
		}
	}

	// create the builder
	log.Infof("Create BuildKit builder with: %s %s", strings.Join(command, " "), strings.Join(args, " "))

	cmd := exec.Command(command[0], completeArgs...)
	// This is necessary because docker would otherwise save the used kube config
	// which we don't want because we will override it with our own temp kube config
	// during building.
	cmd.Env = append(os.Environ(), "KUBECONFIG=")

	out, err := cmd.CombinedOutput()
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
