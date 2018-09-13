package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"context"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/docker/docker/api/types"
	"k8s.io/client-go/tools/clientcmd"
)

var isMinikubeVar *bool

const dockerFileFolder = ".docker"

// Builder holds the necessary information to build and push docker images
type Builder struct {
	RegistryURL string
	ImageName   string
	ImageTag    string
}

// NewBuilder creates a new docker Builder instance
func NewBuilder(registryURL, imageName, imageTag string, preferMinikube bool) *Builder {
	return &Builder{
		RegistryURL: registryURL,
		ImageName:   imageName,
		ImageTag:    imageTag,
	}
}

// Authenticate authenticates the cli with a remote registry
func (b *Builder) Authenticate(user, password string) error {
	return nil
}

// BuildImage builds a dockerimage with the docker cli
func (b *Builder) BuildImage(contextPath, dockerfilePath string, options *types.ImageBuildOptions) error {
	if isMinikube() {
		err := builImageMinikube(dockerfilePath, b.RegistryURL+b.ImageName+b.ImageTag, nil)

		if err == nil {
			return nil
		}

		// Fallback to normal docker cli if minikube failed
	}
	/*
		ctx := context.Background()
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		dockerArgs := []string{"build", cwd, "--file", dockerfilePath, "-t", buildtag}
		dockerArgs = append(dockerArgs, buildArgs...)

		cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

		cmd.Stdout = log.GetInstance()
		cmd.Stderr = log.GetInstance()

		err = cmd.Run()

		if err != nil {
			return err
		}*/

	return nil
}

// PushImage pushes an image to the specified registry
func (b *Builder) PushImage() error {
	/*if isMinikube() {
		err := pushImageMinikube(buildtag)

		if err == nil {
			return nil
		}

		// Fallback to normal docker cli if minikube failed
	}

	ctx := context.Background()
	dockerArgs := []string{"push", buildtag}

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	cmd.Stdout = log.GetInstance()
	cmd.Stderr = log.GetInstance()

	err := cmd.Run()

	if err != nil {
		return err
	}*/

	return nil
}

func pushImageMinikube(buildtag string) error {
	ctx := context.Background()
	dockerArgs, err := getMinikubeCliArgs()
	if err != nil {
		return err
	}

	log.Info("Pushing image on minikube docker daemon")
	dockerArgs = append(dockerArgs, "push", buildtag)

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	cmd.Stdout = log.GetInstance()
	cmd.Stderr = log.GetInstance()

	err = cmd.Run()

	if err != nil {
		return err
	}

	return nil
}

func builImageMinikube(dockerfilePath, buildtag string, buildArgs []string) error {
	ctx := context.Background()
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	dockerArgs, err := getMinikubeCliArgs()
	if err != nil {
		return err
	}

	log.Info("Building image on minikube docker daemon")

	dockerArgs = append(dockerArgs, "build", cwd, "--file", dockerfilePath, "-t", buildtag)
	dockerArgs = append(dockerArgs, buildArgs...)

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	cmd.Stdout = log.GetInstance()
	cmd.Stderr = log.GetInstance()

	err = cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func getMinikubeCliArgs() ([]string, error) {
	env, err := getMinikubeEnvironment()

	if err != nil {
		return nil, err
	}

	dockerArgs := []string{}
	dockerCertPath := env["DOCKER_CERT_PATH"]

	if dockerCertPath != "" {
		dockerArgs = append(dockerArgs, "--tlscacert", filepath.Join(dockerCertPath, "ca.pem"))
		dockerArgs = append(dockerArgs, "--tlscert", filepath.Join(dockerCertPath, "cert.pem"))
		dockerArgs = append(dockerArgs, "--tlskey", filepath.Join(dockerCertPath, "key.pem"))

		if env["DOCKER_TLS_VERIFY"] == "1" {
			dockerArgs = append(dockerArgs, "--tlsverify")
		}
	}

	if env["DOCKER_HOST"] != "" {
		dockerArgs = append(dockerArgs, "--host", env["DOCKER_HOST"])
	} else {
		return nil, fmt.Errorf("Unspecified docker host")
	}

	return dockerArgs, nil
}

func isMinikube() bool {
	if isMinikubeVar == nil {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
		cfg, err := kubeConfig.RawConfig()

		if err != nil {
			return false
		}

		isMinikube := cfg.CurrentContext == "minikube"
		isMinikubeVar = &isMinikube
	}

	return *isMinikubeVar
}

func getMinikubeEnvironment() (map[string]string, error) {
	cmd := exec.Command("minikube", "docker-env", "--shell", "none")
	out, err := cmd.Output()

	if err != nil {
		return nil, err
	}

	env := map[string]string{}

	for _, line := range strings.Split(string(out), "\n") {
		envKeyValue := strings.Split(line, "=")

		if len(envKeyValue) != 2 {
			continue
		}

		env[envKeyValue[0]] = envKeyValue[1]
	}

	return env, nil
}
