package registry

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

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

// InitRegistry deploys and starts a new docker registry if necessary
func InitRegistry(kubectl *kubernetes.Clientset, helm *helm.HelmClientWrapper) error {
	config := configutil.GetConfig(false)
	registryConfig := config.Services.Registry
	registryUser := registryConfig.User
	registryAuthEncoded := base64.StdEncoding.EncodeToString([]byte(*registryUser.Username + ":" + *registryUser.Password))

	if registryConfig.External == nil {
		registry := registryConfig.Internal
		registryReleaseName := *registry.Release.Name
		registryReleaseNamespace := *registry.Release.Namespace

		_, err := helm.InstallChartByName(registryReleaseName, registryReleaseNamespace, "stable/docker-registry", "", registry.Release.Values)

		if err != nil {
			return fmt.Errorf("Unable to initialize docker registry: %s", err.Error())
		}

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
		err = newHtpasswdData.SetPassword(*registryUser.Username, *registryUser.Password, htpasswd.HashBCrypt)

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

		if err != nil {
			return fmt.Errorf("Unable to update htpasswd secret: %s", err.Error())
		}
		registryServiceName := registryReleaseName + "-docker-registry"
		maxServiceWaiting := 60 * time.Second
		serviceWaitingInterval := 3 * time.Second

		for true {
			registryService, err := kubectl.Core().Services(registryReleaseNamespace).Get(registryServiceName, metav1.GetOptions{})

			if err != nil {
				log.Panic(err)
			}

			if len(registryService.Spec.ClusterIP) > 0 {
				registryConfig.Internal.Host = configutil.String(registryService.Spec.ClusterIP + ":" + strconv.Itoa(registryPort))
				break
			}

			time.Sleep(serviceWaitingInterval)
			maxServiceWaiting = maxServiceWaiting - serviceWaitingInterval

			if maxServiceWaiting <= 0 {
				return errors.New("Timeout waiting for registry service to start")
			}
		}
	}
	registryHostname := GetRegistryHostname()

	pullSecretDataValue := []byte(`{
		"auths": {
			"` + registryHostname + `": {
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
	appRelease := config.DevSpace.Release
	appNamespace := *appRelease.Namespace

	_, err := kubectl.Core().Secrets(appNamespace).Get(PullSecretName, metav1.GetOptions{})

	if err != nil {
		_, err = kubectl.Core().Secrets(appNamespace).Create(registryPullSecret)
	} else {
		_, err = kubectl.Core().Secrets(appNamespace).Update(registryPullSecret)
	}

	if err != nil {
		return fmt.Errorf("Unable to update image pull secret: %s", err.Error())
	}

	return nil
}

//GetImageURL returns the image (optional with tag)
func GetImageURL(includingLatestTag bool) string {
	config := configutil.GetConfig(false)
	image := *config.Image.Name

	image = GetRegistryHostname() + "/" + image

	if includingLatestTag {
		image = image + ":" + *config.Image.Tag
	}
	return image
}

//GetRegistryHostname returns the hostname of the registry including the port
func GetRegistryHostname() string {
	config := configutil.GetConfig(false)
	registryConfig := config.Services.Registry

	if registryConfig.External != nil {
		return *registryConfig.External
	}
	registryHostname := ""
	registryReleaseValues := registryConfig.Internal.Release.Values

	if registryReleaseValues != nil {
		registryValues := yamlq.NewQuery(*registryReleaseValues)
		isIngressEnabled, _ := registryValues.Bool("ingress", "enabled")

		if isIngressEnabled {
			firstIngressHostname, _ := registryValues.String("ingress", "hosts", "0")

			if len(firstIngressHostname) > 0 {
				registryHostname = firstIngressHostname
			}
		}
	}

	if len(registryHostname) == 0 {
		registryConfig.Insecure = configutil.Bool(true)
		registryHostname = *registryConfig.Internal.Host
	} else {
		registryConfig.Insecure = configutil.Bool(false)
	}
	return registryHostname
}
