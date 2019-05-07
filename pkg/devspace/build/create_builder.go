package build

import (
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/devspace/builder"
	"github.com/devspace-cloud/devspace/pkg/devspace/builder/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/builder/kaniko"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	dockerclient "github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"k8s.io/client-go/kubernetes"
)

// CreateBuilder creates a new builder
func CreateBuilder(client kubernetes.Interface, imageConfigName string, imageConf *latest.ImageConfig, imageTag string, isDev bool) (builder.Interface, error) {
	var imageBuilder builder.Interface

	if imageConf.Build != nil && imageConf.Build.Custom != nil {

	} else if imageConf.Build != nil && imageConf.Build.Kaniko != nil {
		dockerClient, err := dockerclient.NewClient(false)
		if err != nil {
			return nil, fmt.Errorf("Error creating docker client: %v", err)
		}

		imageBuilder, err = kaniko.NewBuilder(dockerClient, client, imageConfigName, imageConf, imageTag, isDev)
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

		imageBuilder, err = docker.NewBuilder(dockerClient, imageConfigName, imageConf, imageTag, isDev)
		if err != nil {
			return nil, fmt.Errorf("Error creating docker builder: %v", err)
		}
	}

	return imageBuilder, nil
}
