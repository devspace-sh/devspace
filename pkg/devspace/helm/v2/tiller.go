package v2

import (
	"errors"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	helminstaller "k8s.io/helm/cmd/helm/installer"
)

// TillerDeploymentName is the string identifier for the tiller deployment
const TillerDeploymentName = "tiller-deploy"
const stableRepoCachePath = "repository/cache/stable-index.yaml"
const defaultRepositories = `apiVersion: v1
repositories:
- caFile: ""
  cache: ` + stableRepoCachePath + `
  certFile: ""
  keyFile: ""
  name: stable
  url: https://kubernetes-charts.storage.googleapis.com
`

// Ensure that tiller is running
func ensureTiller(config *latest.Config, client kubectl.Client, tillerNamespace string, upgrade bool, log log.Logger) error {
	tillerOptions := getTillerOptions(tillerNamespace)

	// Create tillerNamespace if necessary
	_, err := client.KubeClient().CoreV1().Namespaces().Get(tillerNamespace, metav1.GetOptions{})
	if err != nil {
		log.Donef("Create namespace %s", tillerNamespace)

		// Create tiller namespace
		_, err = client.KubeClient().CoreV1().Namespaces().Create(&k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: tillerNamespace,
			},
		})
		if err != nil {
			return err
		}
	}

	// Create tiller if necessary
	_, err = client.KubeClient().AppsV1().Deployments(tillerNamespace).Get(TillerDeploymentName, metav1.GetOptions{})
	if err != nil {
		// Create tiller server
		err = createTiller(config, client, tillerNamespace, tillerOptions, log)
		if err != nil {
			return err
		}

		log.Done("Tiller started")
	} else if upgrade {
		// Upgrade tiller if necessary
		tillerOptions.ImageSpec = ""
		err = upgradeTiller(client, tillerOptions, log)
		if err != nil {
			return err
		}
	}

	return waitUntilTillerIsStarted(client, tillerNamespace, log)
}

func getTillerOptions(tillerNamespace string) (tillerOptions *helminstaller.Options) {
	return &helminstaller.Options{
		Namespace:                    tillerNamespace,
		MaxHistory:                   10,
		ImageSpec:                    "gcr.io/kubernetes-helm/tiller:v2.16.0",
		ServiceAccount:               TillerServiceAccountName,
		AutoMountServiceAccountToken: true,
	}
}

func createTiller(config *latest.Config, client kubectl.Client, tillerNamespace string, tillerOptions *helminstaller.Options, log log.Logger) error {
	log.StartWait("Installing Tiller server")
	defer log.StopWait()

	// If the service account is already there we do not create it or any roles/rolebindings
	_, err := client.KubeClient().CoreV1().ServiceAccounts(tillerNamespace).Get(TillerServiceAccountName, metav1.GetOptions{})
	if err != nil {
		err = createTillerRBAC(config, client, tillerNamespace, log)
		if err != nil {
			return err
		}
	}

	// Create the deployment
	err = helminstaller.Install(client.KubeClient(), tillerOptions)
	if err != nil {
		return err
	}

	log.Donef("Created deployment %s in %s", TillerDeploymentName, tillerOptions.Namespace)
	return nil
}

func waitUntilTillerIsStarted(client kubectl.Client, tillerNamespace string, log log.Logger) error {
	tillerWaitingTime := 2 * 60 * time.Second
	tillerCheckInterval := 5 * time.Second

	log.StartWait("Waiting for tiller to start")
	defer log.StopWait()

	for tillerWaitingTime > 0 {
		tillerDeployment, err := client.KubeClient().AppsV1().Deployments(tillerNamespace).Get(TillerDeploymentName, metav1.GetOptions{})
		if err != nil {
			continue
		}
		if tillerDeployment.Status.ReadyReplicas == tillerDeployment.Status.Replicas {
			return nil
		}

		time.Sleep(tillerCheckInterval)
		tillerWaitingTime = tillerWaitingTime - tillerCheckInterval
	}

	return errors.New("Tiller didn't start in time")
}

func upgradeTiller(client kubectl.Client, tillerOptions *helminstaller.Options, log log.Logger) error {
	log.StartWait("Upgrading tiller")
	err := helminstaller.Upgrade(client.KubeClient(), tillerOptions)
	log.StopWait()
	if err != nil {
		return err
	}

	return nil
}

// IsTillerDeployed determines if we could connect to a tiller server
func IsTillerDeployed(config *latest.Config, client kubectl.Client, tillerNamespace string) bool {
	deployment, err := client.KubeClient().AppsV1().Deployments(tillerNamespace).Get(TillerDeploymentName, metav1.GetOptions{})
	if err != nil {
		return false
	}

	if deployment == nil {
		return false
	}

	// Check if we have a broken deployment
	if deployment.Status.ReadyReplicas != deployment.Status.Replicas {
		// Delete the tiller deployment
		DeleteTiller(config, client, tillerNamespace)

		return false
	}

	return true
}

// DeleteTiller clears the tiller server, the service account and role binding
func DeleteTiller(config *latest.Config, client kubectl.Client, tillerNamespace string) error {
	propagationPolicy := metav1.DeletePropagationForeground

	// Delete deployment
	client.KubeClient().AppsV1().Deployments(tillerNamespace).Delete(TillerDeploymentName, &metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})

	// Delete service
	client.KubeClient().CoreV1().Services(tillerNamespace).Delete(TillerDeploymentName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})

	// Delete serviceaccount
	client.KubeClient().CoreV1().ServiceAccounts(tillerNamespace).Delete(TillerServiceAccountName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})

	appNamespaces := []string{
		tillerNamespace,
	}

	for _, deployConfig := range config.Deployments {
		if deployConfig.Namespace != "" && deployConfig.Helm != nil {
			appNamespaces = append(appNamespaces, deployConfig.Namespace)
		}
	}

	for _, appNamespace := range appNamespaces {
		client.KubeClient().RbacV1().Roles(appNamespace).Delete(TillerRoleName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
		// client.KubeClient().RbacV1().RoleBindings(*appNamespace).Delete(TillerRoleName+"-binding", &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})

		client.KubeClient().RbacV1().Roles(appNamespace).Delete(TillerRoleManagerName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
		client.KubeClient().RbacV1().RoleBindings(appNamespace).Delete(TillerRoleManagerName+"-binding", &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
	}

	return nil
}
