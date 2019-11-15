package v2

import (
	"regexp"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TillerServiceAccountName is the name of the service account tiller will use
const TillerServiceAccountName = "devspace-tiller"

// TillerRoleName is the name of the role that is assigned to tiller to allow it to deploy to a certain namespace
const TillerRoleName = "devspace-tiller"

// TillerRoleManagerName is the name of the role with minimal rights to allow tiller to manage itself
const TillerRoleManagerName = "tiller-config-manager"

var alreadyExistsRegexp = regexp.MustCompile(".* already exists$")

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
			_, err := client.KubeClient().CoreV1().Namespaces().Get(appNamespace, metav1.GetOptions{})
			if err != nil {
				log.Donef("Create namespace %s", appNamespace)

				_, err = client.KubeClient().CoreV1().Namespaces().Create(&k8sv1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: appNamespace,
					},
				})
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
	_, err := client.KubeClient().CoreV1().ServiceAccounts(tillerNamespace).Create(&k8sv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TillerServiceAccountName,
			Namespace: tillerNamespace,
		},
	})

	return err
}

func addDeployAccessToTiller(client kubectl.Client, tillerNamespace, namespace string) error {
	_, err := client.KubeClient().RbacV1().Roles(namespace).Create(&rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TillerRoleName,
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					rbacv1.APIGroupAll,
					"extensions",
					"apps",
				},
				Resources: []string{rbacv1.ResourceAll},
				Verbs:     []string{rbacv1.ResourceAll},
			},
		},
	})
	if err != nil && alreadyExistsRegexp.Match([]byte(err.Error())) == false {
		return err
	}

	_, err = client.KubeClient().RbacV1().RoleBindings(namespace).Create(&rbacv1.RoleBinding{
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
	})
	if err != nil && alreadyExistsRegexp.Match([]byte(err.Error())) == false {
		return err
	}

	return nil
}
