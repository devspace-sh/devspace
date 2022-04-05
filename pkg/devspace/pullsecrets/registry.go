package pullsecrets

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"time"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/pkg/errors"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PullSecretOptions has all options necessary to create a pullSecret
type PullSecretOptions struct {
	Namespace       string
	RegistryURL     string
	Username        string
	PasswordOrToken string
	Email           string
	Secret          string
}

// DockerConfigJSON represents a local docker auth config file
// for pulling images.
type DockerConfigJSON struct {
	Auths DockerConfig `json:"auths"`
}

// DockerConfig represents the config file used by the docker CLI.
// This config that represents the credentials that should be used
// when pulling images from specific image repositories.
type DockerConfig map[string]DockerConfigEntry

// DockerConfigEntry holds the user information that grant the access to docker registry
type DockerConfigEntry struct {
	Auth  string `json:"auth"`
	Email string `json:"email"`
}

// CreatePullSecret creates an image pull secret for a registry
func (r *client) CreatePullSecret(ctx devspacecontext.Context, options *PullSecretOptions) error {
	pullSecretName := options.Secret
	if pullSecretName == "" {
		pullSecretName = GetRegistryAuthSecretName(options.RegistryURL)
	}

	registryURL := options.RegistryURL
	if registryURL == "hub.docker.com" || registryURL == "" {
		registryURL = "https://index.docker.io/v1/"
	}

	authToken := options.PasswordOrToken
	if options.Username != "" {
		authToken = options.Username + ":" + authToken
	}

	email := options.Email
	if email == "" {
		email = "noreply@devspace.sh"
	}

	err := wait.PollImmediate(time.Second, time.Second*30, func() (bool, error) {
		secret, err := ctx.KubeClient().KubeClient().CoreV1().Secrets(options.Namespace).Get(ctx.Context(), pullSecretName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				// Create the pull secret
				secret, err := newPullSecret(pullSecretName, registryURL, authToken, email)
				if err != nil {
					return false, err
				}

				_, err = ctx.KubeClient().KubeClient().CoreV1().Secrets(options.Namespace).Create(ctx.Context(), secret, metav1.CreateOptions{})
				if err != nil {
					if kerrors.IsAlreadyExists(err) {
						// Retry
						return false, nil
					}

					return false, errors.Wrap(err, "create pull secret")
				}

				ctx.Log().Donef("Created image pull secret %s/%s", options.Namespace, pullSecretName)
				return true, nil
			} else {
				// Retry
				return false, nil
			}
		}

		dockerConfigJSON, err := fromPullSecretData(secret.Data)
		if err != nil {
			return false, err
		}

		existingEntry := dockerConfigJSON.Auths[registryURL]
		updatedEntry := newDockerConfigEntry(authToken, email)
		if hasChanges(existingEntry, updatedEntry) {
			// Update secret entry
			dockerConfigJSON.Auths[registryURL] = updatedEntry

			// Update secret data
			secret.Data, err = toPullSecretData(dockerConfigJSON)
			if err != nil {
				return false, err
			}

			// Update secret
			_, err = ctx.KubeClient().KubeClient().CoreV1().Secrets(options.Namespace).Update(ctx.Context(), secret, metav1.UpdateOptions{})
			if err != nil {
				if kerrors.IsConflict(err) {
					// Retry
					return false, nil
				}

				return false, errors.Wrap(err, "update pull secret")
			}

			ctx.Log().Donef("Updated image pull secret %s/%s", options.Namespace, pullSecretName)
		}
		return true, nil
	})
	if err != nil {
		return errors.Wrap(err, "create pull secret")
	}

	return nil
}

// GetRegistryAuthSecretName returns the name of the image pull secret for a registry
func GetRegistryAuthSecretName(registryURL string) string {
	return "devspace-pull-secrets"
}

func SafeName(name string) string {
	if len(name) > 63 {
		digest := sha256.Sum256([]byte(name))
		return name[0:52] + "-" + hex.EncodeToString(digest[0:])[0:10]
	}
	return name
}

func newPullSecret(name, registryURL, authToken, email string) (*k8sv1.Secret, error) {
	dockerConfig := &DockerConfigJSON{
		Auths: DockerConfig{
			registryURL: newDockerConfigEntry(authToken, email),
		},
	}

	pullSecretData, err := toPullSecretData(dockerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "new pull secret")
	}

	return &k8sv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: pullSecretData,
		Type: k8sv1.SecretTypeDockerConfigJson,
	}, nil
}

func newDockerConfigEntry(authToken, email string) DockerConfigEntry {
	return DockerConfigEntry{
		Auth:  base64.StdEncoding.EncodeToString([]byte(authToken)),
		Email: email,
	}
}

func hasChanges(existing, updated DockerConfigEntry) bool {
	return existing.Auth != updated.Auth || existing.Email != updated.Email
}

func toPullSecretData(dockerConfig *DockerConfigJSON) (map[string][]byte, error) {
	data, err := json.Marshal(dockerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "marshal docker config")
	}

	return map[string][]byte{
		k8sv1.DockerConfigJsonKey: data,
	}, nil
}

func fromPullSecretData(data map[string][]byte) (*DockerConfigJSON, error) {
	dockerConfig := &DockerConfigJSON{}
	if data == nil {
		return dockerConfig, nil
	}

	err := json.Unmarshal(data[k8sv1.DockerConfigJsonKey], &dockerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal docker config")
	}

	return dockerConfig, nil
}
