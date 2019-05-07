package image

import (
	"errors"
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/devspace/builder"
	"github.com/devspace-cloud/devspace/pkg/devspace/builder/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/builder/kaniko"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	dockerclient "github.com/devspace-cloud/devspace/pkg/devspace/docker"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
	"k8s.io/client-go/kubernetes"
)

// CreateBuilder creates a new builder
func CreateBuilder(client kubernetes.Interface, imageConf *latest.ImageConfig, imageTag string, log logpkg.Logger) (builder.Interface, error) {
	config := configutil.GetConfig()
	var imageBuilder builder.Interface

	if imageConf.Build != nil && imageConf.Build.Kaniko != nil {
		buildNamespace, err := configutil.GetDefaultNamespace(config)
		if err != nil {
			return nil, errors.New("Error retrieving default namespace")
		}

		if imageConf.Build.Kaniko.Namespace != nil && *imageConf.Build.Kaniko.Namespace != "" {
			buildNamespace = *imageConf.Build.Kaniko.Namespace
		}

		allowInsecurePush := false
		if imageConf.Insecure != nil {
			allowInsecurePush = *imageConf.Insecure
		}

		pullSecret := ""
		if imageConf.Build.Kaniko.PullSecret != nil {
			pullSecret = *imageConf.Build.Kaniko.PullSecret
		}

		dockerClient, err := dockerclient.NewClient(false)
		if err != nil {
			return nil, fmt.Errorf("Error creating docker client: %v", err)
		}

		imageBuilder, err = kaniko.NewBuilder(pullSecret, *imageConf.Image, imageTag, buildNamespace, imageConf.Build.Kaniko, dockerClient, client, allowInsecurePush, log)
		if err != nil {
			return nil, fmt.Errorf("Error creating kaniko builder: %v", err)
		}
	} else {
		preferMinikube := true
		if imageConf.Build != nil && imageConf.Build.Docker != nil && imageConf.Build.Docker.PreferMinikube != nil {
			preferMinikube = *imageConf.Build.Docker.PreferMinikube
		}

		dockerClient, err := dockerclient.NewClient(preferMinikube)
		if err != nil {
			return nil, fmt.Errorf("Error creating docker client: %v", err)
		}

		imageBuilder, err = docker.NewBuilder(dockerClient, *imageConf.Image, imageTag, log)
		if err != nil {
			return nil, fmt.Errorf("Error creating docker builder: %v", err)
		}
	}

	return imageBuilder, nil
}
