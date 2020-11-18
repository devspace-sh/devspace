package pullsecrets

import (
	"context"
	"encoding/base64"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const registryAuthSecretNamePrefix = "devspace-auth-"

var registryNameReplaceRegex = regexp.MustCompile(`[^a-z0-9\\-]`)

// PullSecretOptions has all options neccessary to create a pullSecret
type PullSecretOptions struct {
	Namespace       string
	RegistryURL     string
	Username        string
	PasswordOrToken string
	Email           string
	Secret          string
}

// CreatePullSecret creates an image pull secret for a registry
func (r *client) CreatePullSecret(options *PullSecretOptions) error {
	pullSecretName := options.Secret
	if pullSecretName == "" {
		pullSecretName = GetRegistryAuthSecretName(options.RegistryURL)
	}

	if options.RegistryURL == "hub.docker.com" || options.RegistryURL == "" {
		options.RegistryURL = "https://index.docker.io/v1/"
	}

	authToken := options.PasswordOrToken
	if options.Username != "" {
		authToken = options.Username + ":" + authToken
	}

	registryAuthEncoded := base64.StdEncoding.EncodeToString([]byte(authToken))
	pullSecretDataValue := []byte(`{
			"auths": {
				"` + options.RegistryURL + `": {
					"auth": "` + registryAuthEncoded + `",
					"email": "` + options.Email + `"
				}
			}
		}`)

	pullSecretData := map[string][]byte{}
	pullSecretDataKey := k8sv1.DockerConfigJsonKey
	pullSecretData[pullSecretDataKey] = pullSecretDataValue

	registryPullSecret := &k8sv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: pullSecretName,
		},
		Data: pullSecretData,
		Type: k8sv1.SecretTypeDockerConfigJson,
	}

	err := wait.PollImmediate(time.Second, time.Second*30, func() (bool, error) {
		secret, err := r.kubeClient.KubeClient().CoreV1().Secrets(options.Namespace).Get(context.TODO(), pullSecretName, metav1.GetOptions{})
		if err != nil {
			_, err = r.kubeClient.KubeClient().CoreV1().Secrets(options.Namespace).Create(context.TODO(), registryPullSecret, metav1.CreateOptions{})
			if err != nil {
				return false, errors.Errorf("Unable to create image pull secret: %s", err.Error())
			}

			r.log.Donef("Created image pull secret %s/%s", options.Namespace, pullSecretName)
		} else if secret.Data == nil || string(secret.Data[pullSecretDataKey]) != string(pullSecretData[pullSecretDataKey]) {
			_, err = r.kubeClient.KubeClient().CoreV1().Secrets(options.Namespace).Update(context.TODO(), registryPullSecret, metav1.UpdateOptions{})
			if err != nil {
				if kerrors.IsConflict(err) {
					return false, nil
				}

				return false, errors.Errorf("Unable to update image pull secret: %s", err.Error())
			}
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
	if registryURL == "" {
		return registryAuthSecretNamePrefix + "docker"
	}

	return registryAuthSecretNamePrefix + registryNameReplaceRegex.ReplaceAllString(strings.ToLower(registryURL), "-")
}
