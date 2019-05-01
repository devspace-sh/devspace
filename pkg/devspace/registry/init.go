package registry

import (
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/docker/docker/client"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"k8s.io/client-go/kubernetes"
)

// InitRegistries initializes all registries
func InitRegistries(dockerClient client.CommonAPIClient, client kubernetes.Interface, log log.Logger) error {
	err := CreatePullSecrets(dockerClient, client, log)
	if err != nil {
		return err
	}

	return nil
}

// CreatePullSecrets creates the image pull secrets
func CreatePullSecrets(dockerClient client.CommonAPIClient, client kubernetes.Interface, log log.Logger) error {
	config := configutil.GetConfig()

	if config.Images != nil {
		for _, imageConf := range *config.Images {
			if imageConf.CreatePullSecret != nil && *imageConf.CreatePullSecret == true {
				registryURL, err := GetRegistryFromImageName(*imageConf.Image)
				if err != nil {
					return err
				}

				log.StartWait("Creating image pull secret for registry: " + registryURL)
				err = createPullSecretForRegistry(dockerClient, client, registryURL, log)
				log.StopWait()
				if err != nil {
					return fmt.Errorf("Failed to create pull secret for registry: %v", err)
				}
			}
		}
	}

	return nil
}

func createPullSecretForRegistry(dockerClient client.CommonAPIClient, client kubernetes.Interface, registryURL string, log log.Logger) error {
	config := configutil.GetConfig()
	defaultNamespace, err := configutil.GetDefaultNamespace(config)
	if err != nil {
		return err
	}

	username, password := "", ""

	if dockerClient != nil {
		authConfig, _ := docker.GetAuthConfig(dockerClient, registryURL, true)
		if authConfig != nil {
			username = authConfig.Username
			password = authConfig.Password
		}
	}

	if config.Deployments != nil && username != "" && password != "" {
		for _, deployConfig := range *config.Deployments {
			email := "noreply@devspace.cloud"

			namespace := defaultNamespace
			if deployConfig.Namespace != nil {
				namespace = *deployConfig.Namespace
			}

			err := CreatePullSecret(client, namespace, registryURL, username, password, email, log)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
