package helm

import (
	"errors"
	"fmt"
	"time"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/log"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

func ensureTiller(kubectlClient *kubernetes.Clientset, config *v1.Config, upgrade bool) error {
	tillerNamespace := *config.Tiller.Namespace
	tillerOptions := &helminstaller.Options{
		Namespace:                    tillerNamespace,
		MaxHistory:                   10,
		ImageSpec:                    "gcr.io/kubernetes-helm/tiller:v2.12.0",
		ServiceAccount:               TillerServiceAccountName,
		AutoMountServiceAccountToken: true,
	}

	_, err := kubectlClient.CoreV1().Namespaces().Get(tillerNamespace, metav1.GetOptions{})
	if err != nil {
		log.Donef("Create namespace %s", tillerNamespace)

		// Create tiller namespace
		_, err = kubectlClient.CoreV1().Namespaces().Create(&k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: tillerNamespace,
			},
		})
		if err != nil {
			return err
		}
	}

	_, err = kubectlClient.ExtensionsV1beta1().Deployments(tillerNamespace).Get(TillerDeploymentName, metav1.GetOptions{})
	if err != nil {
		// Create tiller server
		err = createTiller(kubectlClient, config, tillerOptions)
		if err != nil {
			return err
		}

		log.Done("Tiller started")
	} else if upgrade {
		// Upgrade tiller if necessary
		tillerOptions.ImageSpec = ""
		err = upgradeTiller(kubectlClient, tillerOptions)
		if err != nil {
			return err
		}
	}

	return waitUntilTillerIsStarted(kubectlClient)
}

func createTiller(kubectlClient *kubernetes.Clientset, dsConfig *v1.Config, tillerOptions *helminstaller.Options) error {
	log.StartWait("Installing Tiller server")
	defer log.StopWait()

	// If the service account is already there we do not create it or any roles/rolebindings
	_, err := kubectlClient.CoreV1().ServiceAccounts(*dsConfig.Tiller.Namespace).Get(TillerServiceAccountName, metav1.GetOptions{})
	if err != nil {
		err = createTillerRBAC(kubectlClient, dsConfig)
		if err != nil {
			return err
		}
	}

	// Create the deployment
	err = helminstaller.Install(kubectlClient, tillerOptions)
	if err != nil {
		return err
	}

	log.Donef("Created deployment %s in %s", TillerDeploymentName, tillerOptions.Namespace)
	return nil
}

func waitUntilTillerIsStarted(kubectlClient *kubernetes.Clientset) error {
	tillerWaitingTime := 2 * 60 * time.Second
	tillerCheckInterval := 5 * time.Second
	config := configutil.GetConfig()

	log.StartWait("Waiting for tiller to start")
	defer log.StopWait()

	for tillerWaitingTime > 0 {
		tillerDeployment, err := kubectlClient.ExtensionsV1beta1().Deployments(*config.Tiller.Namespace).Get(TillerDeploymentName, metav1.GetOptions{})
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

func upgradeTiller(kubectlClient *kubernetes.Clientset, tillerOptions *helminstaller.Options) error {
	log.StartWait("Upgrading tiller")
	err := helminstaller.Upgrade(kubectlClient, tillerOptions)
	log.StopWait()
	if err != nil {
		return err
	}

	return nil
}

// IsTillerDeployed determines if we could connect to a tiller server
func IsTillerDeployed(client *kubernetes.Clientset) bool {
	config := configutil.GetConfig()
	tillerNamespace := *config.Tiller.Namespace
	deployment, err := client.ExtensionsV1beta1().Deployments(tillerNamespace).Get(TillerDeploymentName, metav1.GetOptions{})
	if err != nil {
		return false
	}

	if deployment == nil {
		return false
	}

	// Check if we have a broken deployment
	if deployment.Status.ReadyReplicas != deployment.Status.Replicas {
		// Delete the tiller deployment
		DeleteTiller(client)

		return false
	}

	return true
}

// DeleteTiller clears the tiller server, the service account and role binding
func DeleteTiller(kubectlClient *kubernetes.Clientset) error {
	config := configutil.GetConfig()

	tillerNamespace := *config.Tiller.Namespace
	propagationPolicy := metav1.DeletePropagationForeground

	// Delete deployment
	kubectlClient.ExtensionsV1beta1().Deployments(tillerNamespace).Delete(TillerDeploymentName, &metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})

	// Delete service
	kubectlClient.CoreV1().Services(tillerNamespace).Delete(TillerDeploymentName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})

	// Only delete service accounts and roles in non cloud-provider environments
	if config.Cluster.CloudProvider == nil || *config.Cluster.CloudProvider == "" {
		// Delete serviceaccount
		kubectlClient.CoreV1().ServiceAccounts(tillerNamespace).Delete(TillerServiceAccountName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})

		appNamespaces := []*string{
			&tillerNamespace,
		}

		defaultNamespace, err := configutil.GetDefaultNamespace(config)
		if err != nil {
			return fmt.Errorf("Error retrieving default namespace: %v", err)
		}

		if config.InternalRegistry != nil {
			appNamespaces = append(appNamespaces, config.InternalRegistry.Namespace)
		}

		if config.DevSpace.Deployments != nil {
			for _, deployConfig := range *config.DevSpace.Deployments {
				if deployConfig.Namespace != nil && deployConfig.Helm != nil {
					if *deployConfig.Namespace == "" {
						appNamespaces = append(appNamespaces, &defaultNamespace)
						continue
					}

					appNamespaces = append(appNamespaces, deployConfig.Namespace)
				}
			}
		}

		for _, appNamespace := range appNamespaces {
			kubectlClient.RbacV1beta1().Roles(*appNamespace).Delete(TillerRoleName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
			kubectlClient.RbacV1beta1().RoleBindings(*appNamespace).Delete(TillerRoleName+"-binding", &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})

			kubectlClient.RbacV1beta1().Roles(*appNamespace).Delete(TillerRoleManagerName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
			kubectlClient.RbacV1beta1().RoleBindings(*appNamespace).Delete(TillerRoleManagerName+"-binding", &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
		}
	}

	return nil
}
