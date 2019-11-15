package registry

import (
	"encoding/base64"
	"regexp"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const registryAuthSecretNamePrefix = "devspace-auth-"

var registryNameReplaceRegex = regexp.MustCompile(`[^a-z0-9\\-]`)

// CreatePullSecret creates an image pull secret for a registry
func CreatePullSecret(client kubectl.Client, namespace, registryURL, username, passwordOrToken, email string, log log.Logger) error {
	pullSecretName := GetRegistryAuthSecretName(registryURL)
	if registryURL == "hub.docker.com" || registryURL == "" {
		registryURL = "https://index.docker.io/v1/"
	}

	authToken := passwordOrToken
	if username != "" {
		authToken = username + ":" + authToken
	}

	registryAuthEncoded := base64.StdEncoding.EncodeToString([]byte(authToken))
	pullSecretDataValue := []byte(`{
			"auths": {
				"` + registryURL + `": {
					"auth": "` + registryAuthEncoded + `",
					"email": "` + email + `"
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

	secret, err := client.KubeClient().CoreV1().Secrets(namespace).Get(pullSecretName, metav1.GetOptions{})
	if err != nil {
		_, err = client.KubeClient().CoreV1().Secrets(namespace).Create(registryPullSecret)
		if err != nil {
			return errors.Errorf("Unable to create image pull secret: %s", err.Error())
		}

		log.Donef("Created image pull secret %s/%s", namespace, pullSecretName)
	} else if secret.Data == nil || string(secret.Data[pullSecretDataKey]) != string(pullSecretData[pullSecretDataKey]) {
		_, err = client.KubeClient().CoreV1().Secrets(namespace).Update(registryPullSecret)
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
