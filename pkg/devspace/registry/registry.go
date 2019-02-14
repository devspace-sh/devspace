package registry

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/covexo/devspace/pkg/devspace/config/generated"
	v1 "github.com/covexo/devspace/pkg/devspace/config/versions/latest"

	"github.com/covexo/devspace/pkg/util/log"
	"k8s.io/client-go/kubernetes"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const registryAuthSecretNamePrefix = "devspace-registry-auth-"

var pullSecretNames = []string{}

// CreatePullSecret creates an image pull secret for a registry
func CreatePullSecret(kubectl *kubernetes.Clientset, namespace, registryURL, username, passwordOrToken, email string, log log.Logger) error {
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
	_, err := kubectl.Core().Secrets(namespace).Get(pullSecretName, metav1.GetOptions{})

	if err != nil {
		_, err = kubectl.Core().Secrets(namespace).Create(registryPullSecret)
		if err != nil {
			return fmt.Errorf("Unable to create image pull secret: %s", err.Error())
		}

		log.Donef("Created image pull secret %s/%s", namespace, pullSecretName)
	} else {
		_, err = kubectl.Core().Secrets(namespace).Update(registryPullSecret)
		if err != nil {
			return fmt.Errorf("Unable to update image pull secret: %s", err.Error())
		}
	}

	pullSecretNames = append(pullSecretNames, pullSecretName)

	return nil
}

// GetRegistryAuthSecretName returns the name of the image pull secret for a registry
func GetRegistryAuthSecretName(registryURL string) string {
	registryHash := md5.Sum([]byte(registryURL))

	return registryAuthSecretNamePrefix + hex.EncodeToString(registryHash[:])
}

// GetImageWithTag returns the image (optional with tag)
func GetImageWithTag(generatedConfig *generated.Config, imageConfig *v1.ImageConfig, isDev bool) string {
	image := *imageConfig.Name
	if imageConfig.Tag != nil {
		image = image + ":" + *imageConfig.Tag
	} else {
		if isDev {
			image = image + ":" + generatedConfig.GetActive().Dev.ImageTags[image]
		} else {
			image = image + ":" + generatedConfig.GetActive().Deploy.ImageTags[image]
		}
	}

	return image
}

// GetPullSecretNames returns all names of auto-generated image pull secrets
func GetPullSecretNames() []string {
	return pullSecretNames
}
