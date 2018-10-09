package helm

import (
	"errors"
	"strings"
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
	tillerNamespace := *config.Services.Tiller.Release.Namespace
	tillerOptions := &helminstaller.Options{
		Namespace:      tillerNamespace,
		MaxHistory:     10,
		ImageSpec:      "gcr.io/kubernetes-helm/tiller:v2.10.0",
		ServiceAccount: TillerServiceAccountName,
	}

	// Create tiller namespace & ignore any errors
	kubectlClient.CoreV1().Namespaces().Create(&k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: tillerNamespace,
		},
	})

	_, err := kubectlClient.ExtensionsV1beta1().Deployments(tillerNamespace).Get(TillerDeploymentName, metav1.GetOptions{})
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

	return waitUntilTillerIsStarted(kubectlClient, tillerNamespace)
}

func createTiller(kubectlClient *kubernetes.Clientset, dsConfig *v1.Config, tillerOptions *helminstaller.Options) error {
	log.StartWait("Installing Tiller server")
	defer log.StopWait()

	// If the service account is already there we do not create it or any roles/rolebindings
	_, err := kubectlClient.CoreV1().ServiceAccounts(*dsConfig.Services.Tiller.Release.Namespace).Get(TillerServiceAccountName, metav1.GetOptions{})
	if err != nil {
		err = createTillerRBAC(kubectlClient, dsConfig)
		if err != nil {
			return err
		}
	}

	// Create the deployment
	return helminstaller.Install(kubectlClient, tillerOptions)
}

func waitUntilTillerIsStarted(kubectlClient *kubernetes.Clientset, tillerNamespace string) error {
	tillerWaitingTime := 2 * 60 * time.Second
	tillerCheckInterval := 5 * time.Second

	log.StartWait("Waiting for tiller to start")
	defer log.StopWait()

	for tillerWaitingTime > 0 {
		tillerDeployment, err := kubectlClient.ExtensionsV1beta1().Deployments(tillerNamespace).Get(TillerDeploymentName, metav1.GetOptions{})
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

func addAppNamespaces(appNamespaces *[]*string, namespaces []*string) {
	newAppNamespaces := *appNamespaces

	for _, ns := range namespaces {
		isExisting := false

		for _, existingNS := range newAppNamespaces {
			if ns == existingNS {
				isExisting = true
				break
			}
		}

		if !isExisting {
			newAppNamespaces = append(newAppNamespaces, ns)
		}
	}

	appNamespaces = &newAppNamespaces
}

// IsTillerDeployed determines if we could connect to a tiller server
func IsTillerDeployed(kubectlClient *kubernetes.Clientset, tillerConfig *v1.TillerConfig) bool {
	tillerNamespace := *tillerConfig.Release.Namespace
	deployment, err := kubectlClient.ExtensionsV1beta1().Deployments(tillerNamespace).Get(TillerDeploymentName, metav1.GetOptions{})

	if err != nil {
		return false
	}

	if deployment == nil {
		return false
	}

	return true
}

// DeleteTiller clears the tiller server, the service account and role binding
func DeleteTiller(kubectlClient *kubernetes.Clientset) error {
	config := configutil.GetConfig(false)

	tillerConfig := config.Services.Tiller
	tillerNamespace := *tillerConfig.Release.Namespace
	errs := make([]error, 0, 1)
	propagationPolicy := metav1.DeletePropagationForeground

	err := kubectlClient.ExtensionsV1beta1().Deployments(tillerNamespace).Delete(TillerDeploymentName, &metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil && strings.HasSuffix(err.Error(), "not found") == false {
		errs = append(errs, err)
	}

	err = kubectlClient.CoreV1().Services(tillerNamespace).Delete(TillerDeploymentName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
	if err != nil && strings.HasSuffix(err.Error(), "not found") == false {
		errs = append(errs, err)
	}

	// Only delete service accounts and roles in non cloud-provider environments
	if config.Cluster.CloudProvider == nil || *config.Cluster.CloudProvider == "" {
		err = kubectlClient.CoreV1().ServiceAccounts(tillerNamespace).Delete(TillerServiceAccountName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
		if err != nil && strings.HasSuffix(err.Error(), "not found") == false {
			errs = append(errs, err)
		}

		roleNamespace := append(*tillerConfig.AppNamespaces, &tillerNamespace)
		for _, appNamespace := range roleNamespace {
			err = kubectlClient.RbacV1beta1().Roles(*appNamespace).Delete(TillerRoleName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
			if err != nil && strings.HasSuffix(err.Error(), "not found") == false {
				errs = append(errs, err)
			}

			err = kubectlClient.RbacV1beta1().RoleBindings(*appNamespace).Delete(TillerRoleName+"-binding", &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
			if err != nil && strings.HasSuffix(err.Error(), "not found") == false {
				errs = append(errs, err)
			}

			err = kubectlClient.RbacV1beta1().Roles(*appNamespace).Delete(TillerRoleManagerName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
			if err != nil && strings.HasSuffix(err.Error(), "not found") == false {
				errs = append(errs, err)
			}

			err = kubectlClient.RbacV1beta1().RoleBindings(*appNamespace).Delete(TillerRoleManagerName+"-binding", &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
			if err != nil && strings.HasSuffix(err.Error(), "not found") == false {
				errs = append(errs, err)
			}
		}
	}

	// Merge errors
	errorText := ""

	for _, value := range errs {
		errorText += value.Error() + "\n"
	}

	if errorText == "" {
		return nil
	}
	return errors.New(errorText)
}
