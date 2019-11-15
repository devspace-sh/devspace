package registry

import (
	"github.com/pkg/errors"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatePullSecrets creates the image pull secrets
func (r *client) CreatePullSecrets() error {
	if r.config.Images != nil {
		pullSecrets := []string{}
		createPullSecrets := map[string]bool{}

		for _, imageConf := range r.config.Images {
			if imageConf.CreatePullSecret == nil || *imageConf.CreatePullSecret == true {
				registryURL, err := GetRegistryFromImageName(imageConf.Image)
				if err != nil {
					return err
				}

				createPullSecrets[registryURL] = true
			}
		}

		for registryURL := range createPullSecrets {
			displayRegistryURL := registryURL
			if displayRegistryURL == "" {
				displayRegistryURL = "hub.docker.com"
			}

			log.StartWait("Creating image pull secret for registry: " + displayRegistryURL)
			err := r.createPullSecretForRegistry(registryURL)
			log.StopWait()
			if err != nil {
				return errors.Errorf("Failed to create pull secret for registry: %v", err)
			}

			pullSecrets = append(pullSecrets, GetRegistryAuthSecretName(registryURL))
		}

		if len(pullSecrets) > 0 {
			err := r.addPullSecretsToServiceAccount(pullSecrets)
			if err != nil {
				return errors.Wrap(err, "add pull secrets to service account")
			}
		}
	}

	return nil
}

func (r *client) addPullSecretsToServiceAccount(pullSecrets []string) error {
	// Get default service account
	serviceaccount, err := r.kubeClient.KubeClient().CoreV1().ServiceAccounts(r.kubeClient.Namespace()).Get("default", metav1.GetOptions{})
	if err != nil {
		log.Errorf("Couldn't find service account 'default' in namespace '%s': %v", r.kubeClient.Namespace(), err)
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
		_, err := r.kubeClient.KubeClient().CoreV1().ServiceAccounts(r.kubeClient.Namespace()).Update(serviceaccount)
		if err != nil {
			return errors.Wrap(err, "update service account")
		}
	}

	return nil
}

func (r *client) createPullSecretForRegistry(registryURL string) error {
	username, password := "", ""
	if r.dockerClient != nil {
		authConfig, _ := r.dockerClient.GetAuthConfig(registryURL, true)
		if authConfig != nil {
			username = authConfig.Username
			password = authConfig.Password
		}
	}

	if r.config.Deployments != nil && username != "" && password != "" {
		for _, deployConfig := range r.config.Deployments {
			email := "noreply@devspace.cloud"

			namespace := r.kubeClient.Namespace()
			if deployConfig.Namespace != "" {
				namespace = deployConfig.Namespace
			}

			err := r.CreatePullSecret(&PullSecretOptions{
				Namespace:       namespace,
				RegistryURL:     registryURL,
				Username:        username,
				PasswordOrToken: password,
				Email:           email,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
