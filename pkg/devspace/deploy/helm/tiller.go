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
	tillerNamespace := GetTillerNamespace()
	tillerOptions := &helminstaller.Options{
		Namespace:      tillerNamespace,
		MaxHistory:     10,
		ImageSpec:      "gcr.io/kubernetes-helm/tiller:v2.10.0",
		ServiceAccount: TillerServiceAccountName,
	}

	_, err := kubectlClient.CoreV1().Namespaces().Get(tillerNamespace, metav1.GetOptions{})
	if err != nil {
		log.Infof("Create namespace %s", tillerNamespace)

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
	_, err := kubectlClient.CoreV1().ServiceAccounts(GetTillerNamespace()).Get(TillerServiceAccountName, metav1.GetOptions{})
	if err != nil {
		err = createTillerRBAC(kubectlClient, dsConfig)
		if err != nil {
			return err
		}
	}

	// Create the deployment
	return helminstaller.Install(kubectlClient, tillerOptions)
}

func waitUntilTillerIsStarted(kubectlClient *kubernetes.Clientset) error {
	tillerWaitingTime := 2 * 60 * time.Second
	tillerCheckInterval := 5 * time.Second

	log.StartWait("Waiting for tiller to start")
	defer log.StopWait()

	for tillerWaitingTime > 0 {
		tillerDeployment, err := kubectlClient.ExtensionsV1beta1().Deployments(GetTillerNamespace()).Get(TillerDeploymentName, metav1.GetOptions{})
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

// GetTillerNamespace retrieves the tillernamespace
func GetTillerNamespace() string {
	config := configutil.GetConfig()
	if config.DevSpace.Release == nil || config.DevSpace.Release.Namespace == nil {
		log.Panic("Cannot get tiller namespace when helm is not configured")
	}

	if config.Services == nil || config.Services.Tiller == nil || config.Services.Tiller.Release == nil || config.Services.Tiller.Release.Namespace == nil {
		return *config.DevSpace.Release.Namespace
	}

	return *config.Services.Tiller.Release.Namespace
}

// IsTillerDeployed determines if we could connect to a tiller server
func IsTillerDeployed(kubectlClient *kubernetes.Clientset) bool {
	tillerNamespace := GetTillerNamespace()
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
	config := configutil.GetConfig()

	tillerNamespace := GetTillerNamespace()
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

		appNamespaces := []*string{
			config.DevSpace.Release.Namespace,
			&tillerNamespace,
		}

		if config.Services.InternalRegistry != nil && config.Services.InternalRegistry.Release != nil && config.Services.InternalRegistry.Release.Namespace != nil {
			appNamespaces = append(appNamespaces, config.Services.InternalRegistry.Release.Namespace)
		}

		for _, appNamespace := range appNamespaces {
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
