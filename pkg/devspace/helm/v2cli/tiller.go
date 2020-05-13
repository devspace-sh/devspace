package v2cli

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TillerRoleName is the name of the role that is assigned to tiller to allow it to deploy to a certain namespace
const TillerRoleName = "devspace-tiller"

// TillerRoleManagerName is the name of the role with minimal rights to allow tiller to manage itself
const TillerRoleManagerName = "tiller-config-manager"

// TillerServiceAccountName is the name of the service account tiller will use
const TillerServiceAccountName = "devspace-tiller"

// TillerDeploymentName is the string identifier for the tiller deployment
const TillerDeploymentName = "tiller-deploy"

var alreadyExistsRegexp = regexp.MustCompile(".* already exists$")

func (c *client) ensureTiller() error {
	// If the service account is already there we do not create it or any roles/rolebindings
	_, err := c.kubeClient.KubeClient().CoreV1().ServiceAccounts(c.tillerNamespace).Get(context.TODO(), TillerServiceAccountName, metav1.GetOptions{})
	if err != nil {
		err = createTillerRBAC(c.config, c.kubeClient, c.tillerNamespace, c.log)
		if err != nil {
			return err
		}
	}

	args := []string{"init", "--kube-context", c.kubeClient.CurrentContext(), "--tiller-namespace", c.tillerNamespace, "--upgrade", "--service-account", TillerServiceAccountName}
	out, err := c.exec(c.helmPath, args).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error installing tiller: %s => %v", string(out), err)
	}

	return waitUntilTillerIsStarted(c.kubeClient, c.tillerNamespace, c.log)
}

// IsTillerDeployed determines if we could connect to a tiller server
func IsTillerDeployed(config *latest.Config, client kubectl.Client, tillerNamespace string) bool {
	deployment, err := client.KubeClient().AppsV1().Deployments(tillerNamespace).Get(context.TODO(), TillerDeploymentName, metav1.GetOptions{})
	if err != nil {
		return false
	}

	if deployment == nil {
		return false
	}

	return true
}

func waitUntilTillerIsStarted(client kubectl.Client, tillerNamespace string, log log.Logger) error {
	tillerWaitingTime := 2 * 60 * time.Second
	tillerCheckInterval := 5 * time.Second

	log.StartWait("Waiting for tiller to start")
	defer log.StopWait()

	for tillerWaitingTime > 0 {
		tillerDeployment, err := client.KubeClient().AppsV1().Deployments(tillerNamespace).Get(context.TODO(), TillerDeploymentName, metav1.GetOptions{})
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

func createTillerRBAC(config *latest.Config, client kubectl.Client, tillerNamespace string, log log.Logger) error {
	// Create service account
	err := createTillerServiceAccount(client, tillerNamespace)
	if err != nil {
		return err
	}

	// Create cluster role binding if necessary
	err = client.EnsureGoogleCloudClusterRoleBinding(log)
	if err != nil {
		log.Warnf("Couldn't create gke cluster-admin binding: %v", err)
		log.Warnf("This could cause issues with creating the tiller roles")
	}

	// Tiller does need full access to all namespaces is should deploy to and therefore we create the roles & rolebindings
	appNamespaces := []string{tillerNamespace}

	// Add all namespaces that need our permission
	if config.Deployments != nil && len(config.Deployments) > 0 {
		for _, deployConfig := range config.Deployments {
			if deployConfig.Namespace != "" && deployConfig.Helm != nil {
				appNamespaces = append(appNamespaces, deployConfig.Namespace)
			}
		}
	}

	// Add the correct access rights to the tiller server
	for _, appNamespace := range appNamespaces {
		if appNamespace != "default" {
			// Create namespaces if they are not there already
			_, err := client.KubeClient().CoreV1().Namespaces().Get(context.TODO(), appNamespace, metav1.GetOptions{})
			if err != nil {
				log.Donef("Create namespace %s", appNamespace)

				_, err = client.KubeClient().CoreV1().Namespaces().Create(context.TODO(), &k8sv1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: appNamespace,
					},
				}, metav1.CreateOptions{})
				if err != nil {
					return err
				}
			}
		}

		err = addDeployAccessToTiller(client, tillerNamespace, appNamespace)
		if err != nil {
			return err
		}
	}

	return nil
}

func createTillerServiceAccount(client kubectl.Client, tillerNamespace string) error {
	_, err := client.KubeClient().CoreV1().ServiceAccounts(tillerNamespace).Create(context.TODO(), &k8sv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TillerServiceAccountName,
			Namespace: tillerNamespace,
		},
	}, metav1.CreateOptions{})

	return err
}

func addDeployAccessToTiller(client kubectl.Client, tillerNamespace, namespace string) error {
	_, err := client.KubeClient().RbacV1().Roles(namespace).Create(context.TODO(), &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TillerRoleName,
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{rbacv1.APIGroupAll},
				Resources: []string{rbacv1.ResourceAll},
				Verbs:     []string{rbacv1.ResourceAll},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil && alreadyExistsRegexp.Match([]byte(err.Error())) == false {
		return err
	}

	_, err = client.KubeClient().RbacV1().RoleBindings(namespace).Create(context.TODO(), &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TillerRoleName + "-binding",
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      TillerServiceAccountName,
				Namespace: tillerNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     TillerRoleName,
		},
	}, metav1.CreateOptions{})
	if err != nil && alreadyExistsRegexp.Match([]byte(err.Error())) == false {
		return err
	}

	return nil
}
