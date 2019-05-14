package registry

import (
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CreatePullSecrets creates the image pull secrets
func CreatePullSecrets(config *latest.Config, dockerClient client.CommonAPIClient, client kubernetes.Interface, log log.Logger) error {
	if config.Images != nil {
		pullSecrets := []string{}

		for _, imageConf := range *config.Images {
			if imageConf.CreatePullSecret != nil && *imageConf.CreatePullSecret == true {
				registryURL, err := GetRegistryFromImageName(*imageConf.Image)
				if err != nil {
					return err
				}

				log.StartWait("Creating image pull secret for registry: " + registryURL)
				err = createPullSecretForRegistry(config, dockerClient, client, registryURL, log)
				log.StopWait()
				if err != nil {
					return fmt.Errorf("Failed to create pull secret for registry: %v", err)
				}

				pullSecrets = append(pullSecrets, GetRegistryAuthSecretName(registryURL))
			}
		}

		if len(pullSecrets) > 0 {
			err := addPullSecretsToServiceAccount(config, client, pullSecrets, log)
			if err != nil {
				return errors.Wrap(err, "add pull secrets to service account")
			}
		}
	}

	return nil
}

func addPullSecretsToServiceAccount(config *latest.Config, client kubernetes.Interface, pullSecrets []string, log log.Logger) error {
	// Add secrets to default service account in default namespace
	namespace, err := configutil.GetDefaultNamespace(config)
	if err != nil {
		return errors.Wrap(err, "get default namespace")
	}

	// Get default service account
	serviceaccount, err := client.Core().ServiceAccounts(namespace).Get("default", metav1.GetOptions{})
	if err != nil {
		log.Errorf("Couldn't find service account 'default' in namespace '%s': %v", namespace, err)
		return nil
	}

	// Check if all pull secrets are there
	changed := false
	for _, newPullSecret := range pullSecrets {
		found := false

		for _, pullSecret := range serviceaccount.ImagePullSecrets {
			if pullSecret.Name == newPullSecret {
				found = true
				break
			}
		}

		if found == false {
			changed = true
			serviceaccount.ImagePullSecrets = append(serviceaccount.ImagePullSecrets, v1.LocalObjectReference{Name: newPullSecret})
		}
	}

	// Should we update the service account?
	if changed {
		_, err := client.Core().ServiceAccounts(namespace).Update(serviceaccount)
		if err != nil {
			return errors.Wrap(err, "update service account")
		}
	}

	return nil
}

func createPullSecretForRegistry(config *latest.Config, dockerClient client.CommonAPIClient, client kubernetes.Interface, registryURL string, log log.Logger) error {
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
