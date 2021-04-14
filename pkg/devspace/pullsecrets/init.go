package pullsecrets

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"

	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatePullSecrets creates the image pull secrets
func (r *client) CreatePullSecrets() error {
	createPullSecrets := []*latest.PullSecretConfig{}

	// execute before pull secrets hooks
	err := r.hookExecuter.Execute(hook.Before, hook.StagePullSecrets, hook.All, hook.Context{Client: r.kubeClient}, r.log)
	if err != nil {
		return err
	}

	// gather pull secrets from pullSecrets
	for _, pullSecret := range r.config.PullSecrets {
		createPullSecrets = append(createPullSecrets, pullSecret)
	}

	// gather pull secrets from images
	for _, imageConf := range r.config.Images {
		if imageConf.CreatePullSecret == nil || *imageConf.CreatePullSecret == true {
			registryURL, err := GetRegistryFromImageName(imageConf.Image)
			if err != nil {
				return err
			}

			if contains(registryURL, createPullSecrets) == false {
				createPullSecrets = append(createPullSecrets, &latest.PullSecretConfig{
					Registry: registryURL,
				})
			}
		}
	}

	// create pull secrets
	for _, pullSecretConf := range createPullSecrets {
		displayRegistryURL := pullSecretConf.Registry
		if displayRegistryURL == "" {
			displayRegistryURL = "hub.docker.com"
		}
		if pullSecretConf.Secret == "" {
			pullSecretConf.Secret = GetRegistryAuthSecretName(pullSecretConf.Registry)
		}

		r.log.StartWait("Creating image pull secret for registry: " + displayRegistryURL)
		err := r.createPullSecretForRegistry(pullSecretConf)
		r.log.StopWait()
		if err != nil {
			// execute on error pull secrets hooks
			r.hookExecuter.OnError(hook.StagePullSecrets, []string{hook.All}, hook.Context{Client: r.kubeClient, Error: err}, r.log)
			return errors.Errorf("failed to create pull secret for registry: %v", err)
		}

		if len(pullSecretConf.ServiceAccounts) > 0 {
			for _, serviceAccount := range pullSecretConf.ServiceAccounts {
				err = r.addPullSecretsToServiceAccount(pullSecretConf.Secret, serviceAccount)
				if err != nil {
					// execute on error pull secrets hooks
					r.hookExecuter.OnError(hook.StagePullSecrets, []string{hook.All}, hook.Context{Client: r.kubeClient, Error: err}, r.log)
					return errors.Wrap(err, "add pull secrets to service account")
				}
			}
		} else {
			err = r.addPullSecretsToServiceAccount(pullSecretConf.Secret, "default")
			if err != nil {
				// execute on error pull secrets hooks
				r.hookExecuter.OnError(hook.StagePullSecrets, []string{hook.All}, hook.Context{Client: r.kubeClient, Error: err}, r.log)
				return errors.Wrap(err, "add pull secrets to service account")
			}
		}
	}

	// execute after pull secrets hooks
	err = r.hookExecuter.Execute(hook.After, hook.StagePullSecrets, hook.All, hook.Context{Client: r.kubeClient}, r.log)
	if err != nil {
		return err
	}

	return nil
}

func contains(registryURL string, pullSecrets []*latest.PullSecretConfig) bool {
	for _, v := range pullSecrets {
		if v.Registry == registryURL {
			return true
		}
	}
	return false
}

func (r *client) addPullSecretsToServiceAccount(pullSecretName string, serviceAccount string) error {
	if serviceAccount == "" {
		serviceAccount = "default"
	}

	err := wait.PollImmediate(time.Second, time.Second*30, func() (bool, error) {
		// Get default service account
		sa, err := r.kubeClient.KubeClient().CoreV1().ServiceAccounts(r.kubeClient.Namespace()).Get(context.TODO(), serviceAccount, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}

			r.log.Errorf("Couldn't retrieve service account '%s' in namespace '%s': %v", serviceAccount, r.kubeClient.Namespace(), err)
			return false, err
		}

		// Check if all pull secrets are there
		found := false
		for _, pullSecret := range sa.ImagePullSecrets {
			if pullSecret.Name == pullSecretName {
				found = true
				break
			}
		}
		if found == false {
			sa.ImagePullSecrets = append(sa.ImagePullSecrets, v1.LocalObjectReference{Name: pullSecretName})
			_, err := r.kubeClient.KubeClient().CoreV1().ServiceAccounts(r.kubeClient.Namespace()).Update(context.TODO(), sa, metav1.UpdateOptions{})
			if err != nil {
				if kerrors.IsConflict(err) {
					return false, nil
				}

				return false, errors.Wrap(err, "update service account")
			}
		}

		return true, nil
	})
	if err != nil {
		return errors.Wrap(err, "add pull secret to service account")
	}

	return nil
}

func (r *client) createPullSecretForRegistry(pullSecret *latest.PullSecretConfig) error {
	username := pullSecret.Username
	password := pullSecret.Password
	if username == "" && password == "" && r.dockerClient != nil {
		authConfig, _ := r.dockerClient.GetAuthConfig(pullSecret.Registry, true)
		if authConfig != nil {
			username = authConfig.Username
			password = authConfig.Password
		}
	}

	email := pullSecret.Email
	if email == "" {
		email = "noreply@devspace.cloud"
	}

	if username != "" && password != "" {
		defaultNamespace := r.kubeClient.Namespace()
		err := r.CreatePullSecret(&PullSecretOptions{
			Namespace:       defaultNamespace,
			RegistryURL:     pullSecret.Registry,
			Username:        username,
			PasswordOrToken: password,
			Email:           email,
			Secret:          pullSecret.Secret,
		})
		if err != nil {
			return err
		}

		// create pull secrets in other namespaces if there are any
		namespaces := map[string]bool{
			defaultNamespace: true,
		}
		for _, deployConfig := range r.config.Deployments {
			if deployConfig.Namespace == "" || namespaces[deployConfig.Namespace] {
				continue
			}

			err := r.CreatePullSecret(&PullSecretOptions{
				Namespace:       deployConfig.Namespace,
				RegistryURL:     pullSecret.Registry,
				Username:        username,
				PasswordOrToken: password,
				Email:           email,
				Secret:          pullSecret.Secret,
			})
			if err != nil {
				return err
			}

			namespaces[deployConfig.Namespace] = true
		}
	}

	return nil
}
