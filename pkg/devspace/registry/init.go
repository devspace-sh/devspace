package registry

import (
	"errors"
	"fmt"

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

	if registryMap != nil {
		defaultNamespace, err := configutil.GetDefaultNamespace(config)
		if err != nil {
			return fmt.Errorf("Cannot get default namespace: %v", err)
		}

		for registryName, registryConf := range registryMap {
			if registryConf.Auth != nil && registryConf.Auth.Password != nil {
				if config.DevSpace.Deployments != nil {
					for _, deployConfig := range *config.DevSpace.Deployments {
						username := ""
						password := *registryConf.Auth.Password
						email := "noreply@devspace-cloud.com"
						registryURL := ""

						if registryConf.Auth.Username != nil {
							username = *registryConf.Auth.Username
						}
						if registryConf.URL != nil {
							registryURL = *registryConf.URL
						}

						namespace := *deployConfig.Namespace
						if namespace == "" {
							namespace = defaultNamespace
						}

						log.StartWait("Creating image pull secret for registry: " + registryName)
						err := CreatePullSecret(client, namespace, registryURL, username, password, email)
						log.StopWait()

						if err != nil {
							return fmt.Errorf("Failed to create pull secret for registry: %v", err)
						}
					}
				}
			}
		}
	}

	return nil
}
