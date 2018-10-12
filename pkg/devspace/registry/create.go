package registry

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/devspace/deploy/helm"
	"github.com/covexo/yamlq"
	"github.com/foomo/htpasswd"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createRegistry(kubectl *kubernetes.Clientset, helm *helm.ClientWrapper, internalRegistry *v1.InternalRegistry, registryConfig *v1.RegistryConfig) error {
	registryReleaseName := *internalRegistry.Release.Name
	registryReleaseNamespace := *internalRegistry.Release.Namespace
	registryReleaseValues := internalRegistry.Release.Values

	_, err := kubectl.CoreV1().Namespaces().Get(registryReleaseNamespace, metav1.GetOptions{})
	if err != nil {
		// Create registryReleaseNamespace
		_, err = kubectl.CoreV1().Namespaces().Create(&k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: registryReleaseNamespace,
			},
		})
		if err != nil {
			return err
		}
	}

	// Deploy the registry
	_, err = helm.InstallChartByName(registryReleaseName, registryReleaseNamespace, "stable/docker-registry", "", registryReleaseValues)
	if err != nil {
		return fmt.Errorf("Unable to initialize docker registry: %s", err.Error())
	}

	// Create/Update secret if necessary
	if registryConfig != nil && registryConfig.Auth != nil {
		// Update registry secret
		err = createOrUpdateRegistrySecret(kubectl, internalRegistry, registryConfig)
		if err != nil {
			return err
		}
	}

	// Get the registry url
	serviceHostname, err := getRegistryURL(kubectl, registryReleaseNamespace, registryReleaseName+"-docker-registry")
	if err != nil {
		return err
	}

	// Check if an ingress is configured
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

	// Update config values
	if len(ingressHostname) == 0 {
		registryConfig.URL = configutil.String(serviceHostname)
		registryConfig.Insecure = configutil.Bool(true)
	} else {
		registryConfig.URL = configutil.String(ingressHostname)
		registryConfig.Insecure = configutil.Bool(false)
	}

	return nil
}

func createOrUpdateRegistrySecret(kubectl *kubernetes.Clientset, internalRegistry *v1.InternalRegistry, registryConfig *v1.RegistryConfig) error {
	registryReleaseName := *internalRegistry.Release.Name
	registryReleaseNamespace := *internalRegistry.Release.Namespace

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

	if err != nil {
		return fmt.Errorf("Unable to update htpasswd secret: %s", err.Error())
	}

	return nil
}

func getRegistryURL(kubectl *kubernetes.Clientset, registryReleaseNamespace, registryServiceName string) (string, error) {
	maxServiceWaiting := 60 * time.Second
	serviceWaitingInterval := 3 * time.Second

	for true {
		registryService, err := kubectl.Core().Services(registryReleaseNamespace).Get(registryServiceName, metav1.GetOptions{})
		if err != nil {
			return "", err
		}

		if len(registryService.Spec.ClusterIP) > 0 {
			return registryService.Spec.ClusterIP + ":" + strconv.Itoa(registryPort), nil
		}

		time.Sleep(serviceWaitingInterval)
		maxServiceWaiting = maxServiceWaiting - serviceWaitingInterval

		if maxServiceWaiting <= 0 {
			return "", errors.New("Timeout waiting for registry service to start")
		}
	}

	return "", nil
}
