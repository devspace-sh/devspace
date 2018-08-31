package registry

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/randutil"
	"github.com/foomo/htpasswd"
	"k8s.io/client-go/kubernetes"

	"github.com/covexo/devspace/pkg/devspace/clients/helm"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PullSecretName for the docker registry
const PullSecretName = "devspace-pull-secret"

// InitDockerRegistry deploys and starts a new docker registry if necessary
func InitDockerRegistry(kubectl *kubernetes.Clientset, helm *helm.HelmClientWrapper, privateConfig *v1.PrivateConfig, dsConfig *v1.DevSpaceConfig) (string, string, error) {
	registryReleaseName := privateConfig.Registry.Release.Name
	registryReleaseNamespace := privateConfig.Registry.Release.Namespace
	registryConfig := dsConfig.Registry
	registrySecrets, secretsExist := registryConfig["secrets"]

	if !secretsExist {
		//TODO
	}
	_, secretIsMap := registrySecrets.(map[interface{}]interface{})

	if !secretIsMap {
		//TODO
	}
	_, err := helm.InstallChartByName(registryReleaseName, registryReleaseNamespace, "stable/docker-registry", "", &registryConfig)

	if err != nil {
		return "", "", fmt.Errorf("Unable to initialize docker registry: %s", err.Error())
	}

	htpasswdSecretName := registryReleaseName + "-docker-registry-secret"
	htpasswdSecret, err := kubectl.Core().Secrets(registryReleaseNamespace).Get(htpasswdSecretName, metav1.GetOptions{})

	if err != nil {
		return "", "", fmt.Errorf("Unable to retrieve secret for docker registry: %s", err.Error())
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

	registryUser := privateConfig.Registry.User
	err = newHtpasswdData.SetPassword(registryUser.Username, registryUser.Password, htpasswd.HashBCrypt)

	if err != nil {
		return "", "", fmt.Errorf("Unable to set password in htpasswd: %s", err.Error())
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
		return "", "", fmt.Errorf("Unable to update htpasswd secret: %s", err.Error())
	}

	registryAuthEncoded := base64.StdEncoding.EncodeToString([]byte(privateConfig.Registry.User.Username + ":" + privateConfig.Registry.User.Password))
	registryServiceName := registryReleaseName + "-docker-registry"

	var registryService *k8sv1.Service

	maxServiceWaiting := 60 * time.Second
	serviceWaitingInterval := 3 * time.Second

	for true {
		registryService, err = kubectl.Core().Services(registryReleaseNamespace).Get(registryServiceName, metav1.GetOptions{})

		if err != nil {
			log.Panic(err)
		}

		if len(registryService.Spec.ClusterIP) > 0 {
			break
		}

		time.Sleep(serviceWaitingInterval)
		maxServiceWaiting = maxServiceWaiting - serviceWaitingInterval

		if maxServiceWaiting <= 0 {
			return "", "", errors.New("Timeout waiting for registry service to start")
		}
	}

	registryPort := 5000
	registryIP := registryService.Spec.ClusterIP + ":" + strconv.Itoa(registryPort)
	registryHostname := registryServiceName + "." + registryReleaseNamespace + ".svc.cluster.local:" + strconv.Itoa(registryPort)
	latestImageTag, _ := randutil.GenerateRandomString(10)

	latestImageHostname := registryHostname + "/" + privateConfig.Release.Name + ":" + latestImageTag
	latestImageIP := registryIP + "/" + privateConfig.Release.Name + ":" + latestImageTag

	pullSecretDataValue := []byte(`{
		"auths": {
			"` + registryHostname + `": {
				"auth": "` + registryAuthEncoded + `",
				"email": "noreply-devspace@covexo.com"
			},
			
			"` + registryIP + `": {
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

	_, err = kubectl.Core().Secrets(privateConfig.Release.Namespace).Get(PullSecretName, metav1.GetOptions{})

	if err != nil {
		_, err = kubectl.Core().Secrets(privateConfig.Release.Namespace).Create(registryPullSecret)
	} else {
		_, err = kubectl.Core().Secrets(privateConfig.Release.Namespace).Update(registryPullSecret)
	}

	if err != nil {
		return "", "", fmt.Errorf("Unable to update image pull secret: %s", err.Error())
	}

	return latestImageHostname, latestImageIP, nil
}
