package pullsecrets

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *client) EnsurePullSecret(ctx *devspacecontext.Context, namespace, registryURL string) error {
	pullSecret := &latest.PullSecretConfig{Registry: registryURL}

	// try to find in pull secrets
	if ctx.Config != nil && ctx.Config.Config() != nil {
		for _, ps := range ctx.Config.Config().PullSecrets {
			if ps.Registry == registryURL {
				pullSecret = ps
				break
			}
		}
	}

	return r.ensurePullSecret(ctx, namespace, pullSecret)
}

func (r *client) ensurePullSecret(ctx *devspacecontext.Context, namespace string, pullSecretConf *latest.PullSecretConfig) error {
	if pullSecretConf.Disabled {
		return nil
	}

	displayRegistryURL := pullSecretConf.Registry
	if displayRegistryURL == "" {
		displayRegistryURL = "hub.docker.com"
	}
	if pullSecretConf.Secret == "" {
		pullSecretConf.Secret = GetRegistryAuthSecretName(pullSecretConf.Registry)
	}

	ctx.Log.Info("Ensuring image pull secret for registry: " + displayRegistryURL + "...")
	err := r.createPullSecret(ctx, pullSecretConf)
	if err != nil {
		return errors.Errorf("failed to create pull secret for registry: %v", err)
	}

	if len(pullSecretConf.ServiceAccounts) > 0 {
		for _, serviceAccount := range pullSecretConf.ServiceAccounts {
			err = r.addPullSecretsToServiceAccount(ctx, namespace, pullSecretConf.Secret, serviceAccount)
			if err != nil {
				return errors.Wrap(err, "add pull secrets to service account")
			}
		}
	} else {
		err = r.addPullSecretsToServiceAccount(ctx, namespace, pullSecretConf.Secret, "default")
		if err != nil {
			return errors.Wrap(err, "add pull secrets to service account")
		}
	}

	return nil
}

// EnsurePullSecrets creates the image pull secrets
func (r *client) EnsurePullSecrets(ctx *devspacecontext.Context, namespace string) (err error) {
	createPullSecrets := []*latest.PullSecretConfig{}

	// gather pull secrets from pullSecrets
	if ctx.Config != nil {
		createPullSecrets = append(createPullSecrets, ctx.Config.Config().PullSecrets...)
	}

	defer func() {
		if err != nil {
			// execute on error pull secrets hooks
			pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
				"PULL_SECRETS": createPullSecrets,
				"error":        err,
			}, "error:createPullSecrets")
			if pluginErr != nil {
				return
			}
		}
	}()

	// execute before pull secrets hooks
	pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
		"PULL_SECRETS": createPullSecrets,
	}, "before:createPullSecrets")
	if pluginErr != nil {
		return pluginErr
	}

	// create pull secrets
	for _, pullSecretConf := range createPullSecrets {
		err = r.ensurePullSecret(ctx, namespace, pullSecretConf)
		if err != nil {
			return err
		}
	}

	// execute after pull secrets hooks
	pluginErr = hook.ExecuteHooks(ctx, map[string]interface{}{
		"PULL_SECRETS": createPullSecrets,
	}, "after:createPullSecrets")
	if pluginErr != nil {
		return pluginErr
	}

	return nil
}

func (r *client) addPullSecretsToServiceAccount(ctx *devspacecontext.Context, namespace, pullSecretName string, serviceAccount string) error {
	if serviceAccount == "" {
		serviceAccount = "default"
	}

	err := wait.PollImmediate(time.Second, time.Second*30, func() (bool, error) {
		// Get default service account
		sa, err := ctx.KubeClient.KubeClient().CoreV1().ServiceAccounts(namespace).Get(ctx.Context, serviceAccount, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}

			ctx.Log.Errorf("Couldn't retrieve service account '%s' in namespace '%s': %v", serviceAccount, namespace, err)
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
		if !found {
			sa.ImagePullSecrets = append(sa.ImagePullSecrets, v1.LocalObjectReference{Name: pullSecretName})
			_, err := ctx.KubeClient.KubeClient().CoreV1().ServiceAccounts(namespace).Update(ctx.Context, sa, metav1.UpdateOptions{})
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

func (r *client) createPullSecret(ctx *devspacecontext.Context, pullSecret *latest.PullSecretConfig) error {
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
		err := r.CreatePullSecret(ctx, &PullSecretOptions{
			Namespace:       ctx.KubeClient.Namespace(),
			RegistryURL:     pullSecret.Registry,
			Username:        username,
			PasswordOrToken: password,
			Email:           email,
			Secret:          pullSecret.Secret,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
