package registry

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/covexo/devspace/pkg/devspace/config/v1"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/foomo/htpasswd"
	"k8s.io/client-go/kubernetes"

	"github.com/covexo/devspace/pkg/devspace/clients/helm"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/yamlq"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PullSecretName for the docker registry
const PullSecretName = "devspace-pull-secret"
const registryPort = 5000

// CreatePullSecret creates an image pull secret for a registry
func CreatePullSecret(kubectl *kubernetes.Clientset, namespace string, registryConfig *v1.RegistryConfig) error {
	registryAuth := registryConfig.Auth

	if registryAuth != nil && registryAuth.Username != nil && registryAuth.Password != nil {
		registryAuthEncoded := base64.StdEncoding.EncodeToString([]byte(*registryAuth.Username + ":" + *registryAuth.Password))
		pullSecretDataValue := []byte(`{
		"auths": {
			"` + *registryConfig.URL + `": {
				"auth": "` + registryAuthEncoded + `",
				"email": "noreply-devspace@covexo.com"
			}
		}
	}`)

		pullSecretData := map[string][]byte{}
		pullSecretDataKey := k8sv1.DockerConfigJsonKey
		pullSecretData[pullSecretDataKey] = pullSecretDataValue

		registryPullSecret := &k8sv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: PullSecretName,
			},
			Data: pullSecretData,
			Type: k8sv1.SecretTypeDockerConfigJson,
		}
		_, err := kubectl.Core().Secrets(namespace).Get(PullSecretName, metav1.GetOptions{})

		if err != nil {
			_, err = kubectl.Core().Secrets(namespace).Create(registryPullSecret)
		} else {
			_, err = kubectl.Core().Secrets(namespace).Update(registryPullSecret)
		}

		if err != nil {
			return fmt.Errorf("Unable to update image pull secret: %s", err.Error())
		}
	}
	return nil
}

// InitInternalRegistry deploys and starts a new docker registry if necessary
func InitInternalRegistry(kubectl *kubernetes.Clientset, helm *helm.HelmClientWrapper, internalRegistry *v1.InternalRegistry, registryConfig *v1.RegistryConfig) error {
	registryReleaseName := *internalRegistry.Release.Name
	registryReleaseNamespace := *internalRegistry.Release.Namespace
	registryReleaseValues := internalRegistry.Release.Values

	// Check if registry namespace exists
	_, err := kubectl.CoreV1().Namespaces().Get(registryReleaseNamespace, metav1.GetOptions{})
	if err != nil {
		// Create registry namespace
		_, err = kubectl.CoreV1().Namespaces().Create(&k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: registryReleaseNamespace,
			},
		})

		if err != nil {
			return err
		}
	}

	_, err = helm.InstallChartByName(registryReleaseName, registryReleaseNamespace, "stable/docker-registry", "", registryReleaseValues)

	if err != nil {
		return fmt.Errorf("Unable to initialize docker registry: %s", err.Error())
	}

	if registryConfig != nil && registryConfig.Auth != nil {
		registryAuth := registryConfig.Auth
		htpasswdSecretName := registryReleaseName + "-docker-registry-secret"
		htpasswdSecret, err := kubectl.Core().Secrets(registryReleaseNamespace).Get(htpasswdSecretName, metav1.GetOptions{})

		if err != nil {
			return fmt.Errorf("Unable to retrieve secret for docker registry: %s", err.Error())
		}

		if htpasswdSecret == nil || htpasswdSecret.Data == nil {
			htpasswdSecret = &k8sv1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: htpasswdSecretName,
				},
				Data: map[string][]byte{},
			}
		}

		oldHtpasswdData := htpasswdSecret.Data["htpasswd"]
		newHtpasswdData := htpasswd.HashedPasswords{}

		if len(oldHtpasswdData) != 0 {
			oldHtpasswdDataBytes := []byte(oldHtpasswdData)
			newHtpasswdData, _ = htpasswd.ParseHtpasswd(oldHtpasswdDataBytes)
		}
		err = newHtpasswdData.SetPassword(*registryAuth.Username, *registryAuth.Password, htpasswd.HashBCrypt)

		if err != nil {
			return fmt.Errorf("Unable to set password in htpasswd: %s", err.Error())
		}

		newHtpasswdDataBytes := newHtpasswdData.Bytes()

		htpasswdSecret.Data["htpasswd"] = newHtpasswdDataBytes

		_, err = kubectl.Core().Secrets(registryReleaseNamespace).Get(htpasswdSecretName, metav1.GetOptions{})

		if err != nil {
			_, err = kubectl.Core().Secrets(registryReleaseNamespace).Create(htpasswdSecret)
		} else {
			_, err = kubectl.Core().Secrets(registryReleaseNamespace).Update(htpasswdSecret)
		}
	}

	if err != nil {
		return fmt.Errorf("Unable to update htpasswd secret: %s", err.Error())
	}
	registryServiceName := registryReleaseName + "-docker-registry"
	serviceHostname := ""
	maxServiceWaiting := 60 * time.Second
	serviceWaitingInterval := 3 * time.Second

	for true {
		registryService, err := kubectl.Core().Services(registryReleaseNamespace).Get(registryServiceName, metav1.GetOptions{})

		if err != nil {
			log.Panic(err)
		}

		if len(registryService.Spec.ClusterIP) > 0 {
			serviceHostname = registryService.Spec.ClusterIP + ":" + strconv.Itoa(registryPort)
			break
		}

		time.Sleep(serviceWaitingInterval)
		maxServiceWaiting = maxServiceWaiting - serviceWaitingInterval

		if maxServiceWaiting <= 0 {
			return errors.New("Timeout waiting for registry service to start")
		}
	}
	ingressHostname := ""

	if registryReleaseValues != nil {
		registryValues := yamlq.NewQuery(*registryReleaseValues)
		isIngressEnabled, _ := registryValues.Bool("ingress", "enabled")

		if isIngressEnabled {
			firstIngressHostname, _ := registryValues.String("ingress", "hosts", "0")

			if len(firstIngressHostname) > 0 {
				ingressHostname = firstIngressHostname
			}
		}
	}

	if len(ingressHostname) == 0 {
		registryConfig.URL = configutil.String(serviceHostname)
		registryConfig.Insecure = configutil.Bool(true)
	} else {
		registryConfig.URL = configutil.String(ingressHostname)
		registryConfig.Insecure = configutil.Bool(false)
	}
	return nil
}

//GetImageURL returns the image (optional with tag)
func GetImageURL(imageConfig *v1.ImageConfig, includingLatestTag bool) string {
	registryConfig, registryConfErr := GetRegistryConfig(imageConfig)

	if registryConfErr != nil {
		log.Fatal(registryConfErr)
	}
	image := *imageConfig.Name
	registryURL := *registryConfig.URL

	if registryURL != "" && registryURL != "hub.docker.com" {
		image = registryURL + "/" + image
	}

	if includingLatestTag {
		image = image + ":" + *imageConfig.Tag
	}
	return image
}

// GetRegistryConfig returns the registry config for an image or an error if the registry is not defined
func GetRegistryConfig(imageConfig *v1.ImageConfig) (*v1.RegistryConfig, error) {
	config := configutil.GetConfig(false)
	registryName := *imageConfig.Registry
	registryMap := *config.Registries
	registryConfig, registryFound := registryMap[registryName]

	if !registryFound {
		return nil, errors.New("Unable to find registry: " + registryName)
	}
	return registryConfig, nil
}
