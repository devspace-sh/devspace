package registry

import (
	"errors"
	"fmt"

	"github.com/covexo/devspace/pkg/devspace/builder/docker"
	"github.com/covexo/devspace/pkg/devspace/config/v1"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/helm"
	"github.com/covexo/devspace/pkg/util/log"
	"k8s.io/client-go/kubernetes"
)

// InitRegistries initializes all registries
func InitRegistries(client *kubernetes.Clientset, log log.Logger) error {
	config := configutil.GetConfig()
	registryMap := *config.Registries

	if config.InternalRegistry != nil && config.InternalRegistry.Deploy != nil && *config.InternalRegistry.Deploy == true {
		registryConf, regConfExists := registryMap["internal"]
		if !regConfExists {
			return errors.New("Registry config not found for internal registry")
		}

		log.StartWait("Initializing helm client")
		helm, err := helm.NewClient(client, log, false)
		log.StopWait()
		if err != nil {
			return fmt.Errorf("Error initializing helm client: %v", err)
		}

		log.StartWait("Initializing internal registry")
		err = InitInternalRegistry(client, helm, config.InternalRegistry, registryConf)
		log.StopWait()
		if err != nil {
			return fmt.Errorf("Internal registry error: %v", err)
		}

		log.Done("Internal registry started")
	}

	mustInitDockerHub := true

	if registryMap != nil {
		for registryName, registryConf := range registryMap {
			if *registryConf.URL == "hub.docker.com" || *registryConf.URL == "index.docker.io" {
				mustInitDockerHub = false
			}

			log.StartWait("Creating image pull secret for registry: " + registryName)
			err := initRegistry(client, registryConf)
			log.StopWait()

			if err != nil {
				return fmt.Errorf("Failed to create pull secret for registry: %v", err)
			}
		}
	}

	if mustInitDockerHub {
		log.StartWait("Creating image pull secret for registry: hub.docker.com")
		err := initRegistry(client, &v1.RegistryConfig{
			URL: configutil.String(""),
		})
		log.StopWait()

		if err != nil {
			return fmt.Errorf("Failed to create pull secret for registry: %v", err)
		}
	}

	return nil
}

func initRegistry(client *kubernetes.Clientset, registryConf *v1.RegistryConfig) error {
	config := configutil.GetConfig()
	defaultNamespace, err := configutil.GetDefaultNamespace(config)
	if err != nil {
		return err
	}

	registryURL := ""
	if registryConf.URL != nil {
		registryURL = *registryConf.URL
	}

	username := ""
	password := ""

	if registryConf.Auth == nil || registryConf.Auth.Username == nil || registryConf.Auth.Password == nil {
		authConfig, _ := docker.GetAuthConfig(registryURL)

		if authConfig != nil {
			username = authConfig.Username
			password = authConfig.Password
		}
	}

	if registryConf.Auth != nil {
		if registryConf.Auth.Username != nil {
			username = *registryConf.Auth.Username
		}

		if registryConf.Auth.Password != nil {
			password = *registryConf.Auth.Password
		}
	}

	if config.DevSpace.Deployments != nil {
		for _, deployConfig := range *config.DevSpace.Deployments {
			email := "noreply@devspace-cloud.com"

			namespace := *deployConfig.Namespace
			if namespace == "" {
				namespace = defaultNamespace
			}

			err := CreatePullSecret(client, namespace, registryURL, username, password, email)

			if err != nil {
				return err
			}
		}
	}
	return nil
}
