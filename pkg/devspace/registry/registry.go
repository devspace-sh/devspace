package registry

import (
	"encoding/base64"
	"regexp"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const registryAuthSecretNamePrefix = "devspace-auth-"

var registryNameReplaceRegex = regexp.MustCompile(`[^a-z0-9\\-]`)

// PullSecretOptions has all options neccessary to create a pullSecret
type PullSecretOptions struct {
	Namespace, RegistryURL, Username, PasswordOrToken, Email string
}

// CreatePullSecret creates an image pull secret for a registry
func (r *client) CreatePullSecret(options *PullSecretOptions) error {
	pullSecretName := GetRegistryAuthSecretName(options.RegistryURL)
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

	secret, err := r.kubeClient.KubeClient().CoreV1().Secrets(options.Namespace).Get(pullSecretName, metav1.GetOptions{})
	if err != nil {
		_, err = r.kubeClient.KubeClient().CoreV1().Secrets(options.Namespace).Create(registryPullSecret)
		if err != nil {
			return errors.Errorf("Unable to create image pull secret: %s", err.Error())
		}

		log.Donef("Created image pull secret %s/%s", options.Namespace, pullSecretName)
	} else if secret.Data == nil || string(secret.Data[pullSecretDataKey]) != string(pullSecretData[pullSecretDataKey]) {
		_, err = r.kubeClient.KubeClient().CoreV1().Secrets(options.Namespace).Update(registryPullSecret)
		if err != nil {
			return errors.Errorf("Unable to update image pull secret: %s", err.Error())
		}
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
