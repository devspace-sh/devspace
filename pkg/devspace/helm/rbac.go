package helm

import (
	"regexp"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"

	"github.com/covexo/devspace/pkg/util/log"
	k8sv1 "k8s.io/api/core/v1"
	k8sv1beta1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// TillerServiceAccountName is the name of the service account tiller will use
const TillerServiceAccountName = "devspace-tiller"

// TillerRoleName is the name of the role that is assigned to tiller to allow it to deploy to a certain namespace
const TillerRoleName = "devspace-tiller"

// TillerRoleManagerName is the name of the role with minimal rights to allow tiller to manage itself
const TillerRoleManagerName = "tiller-config-manager"

var alreadyExistsRegexp = regexp.MustCompile(".* already exists$")

func createTillerRBAC(kubectlClient *kubernetes.Clientset, tillerNamespace string) error {
	config := configutil.GetConfig()

	// Create service account
	err := createTillerServiceAccount(kubectlClient, tillerNamespace)
	if err != nil {
		return err
	}

	// Tiller does need full access to all namespaces is should deploy to and therefore we create the roles & rolebindings
	appNamespaces := []*string{&tillerNamespace}

	// Get default namespace
	defaultNamespace, err := configutil.GetDefaultNamespace(config)
	if err != nil {
		return err
	}

	// Add all namespaces that need our permission
	if config.Deployments != nil && len(*config.Deployments) > 0 {
		for _, deployConfig := range *config.Deployments {
			if deployConfig.Namespace != nil && deployConfig.Helm != nil {
				if *deployConfig.Namespace == "" {
					appNamespaces = append(appNamespaces, &defaultNamespace)
					continue
				}

				appNamespaces = append(appNamespaces, deployConfig.Namespace)
			}
		}
	}

	// Add the correct access rights to the tiller server
	for _, appNamespace := range appNamespaces {
		if *appNamespace != "default" {
			// Create namespaces if they are not there already
			_, err := kubectlClient.CoreV1().Namespaces().Get(*appNamespace, metav1.GetOptions{})
			if err != nil {
				log.Donef("Create namespace %s", *appNamespace)

				_, err = kubectlClient.CoreV1().Namespaces().Create(&k8sv1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: *appNamespace,
					},
				})
				if err != nil {
					return err
				}
			}
		}

		err = addDeployAccessToTiller(kubectlClient, tillerNamespace, *appNamespace)
		if err != nil {
			return err
		}
	}

	return nil
}

func createTillerServiceAccount(kubectlClient *kubernetes.Clientset, tillerNamespace string) error {
	_, err := kubectlClient.CoreV1().ServiceAccounts(tillerNamespace).Create(&k8sv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TillerServiceAccountName,
			Namespace: tillerNamespace,
		},
	})

	return err
}

func addMinimalAccessToTiller(kubectlClient *kubernetes.Clientset, tillerNamespace string) error {
	_, err := kubectlClient.RbacV1beta1().Roles(tillerNamespace).Create(&k8sv1beta1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TillerRoleManagerName,
			Namespace: tillerNamespace,
		},
		Rules: []k8sv1beta1.PolicyRule{
			{
				APIGroups: []string{
					k8sv1beta1.APIGroupAll,
					"extensions",
					"apps",
				},
				Resources: []string{
					"configmaps",
				},
				Verbs: []string{k8sv1beta1.ResourceAll},
			},
		},
	})
	if err != nil && alreadyExistsRegexp.Match([]byte(err.Error())) == false {
		return err
	}

	_, err = kubectlClient.RbacV1beta1().RoleBindings(tillerNamespace).Create(&k8sv1beta1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TillerRoleManagerName + "-binding",
			Namespace: tillerNamespace,
		},
		Subjects: []k8sv1beta1.Subject{
			{
				Kind:      k8sv1beta1.ServiceAccountKind,
				Name:      TillerServiceAccountName,
				Namespace: tillerNamespace,
			},
		},
		RoleRef: k8sv1beta1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     TillerRoleManagerName,
		},
	})
	if err != nil && alreadyExistsRegexp.Match([]byte(err.Error())) == false {
		return err
	}

	return nil
}

func addDeployAccessToTiller(kubectlClient *kubernetes.Clientset, tillerNamespace, namespace string) error {
	_, err := kubectlClient.RbacV1beta1().Roles(namespace).Create(&k8sv1beta1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TillerRoleName,
			Namespace: namespace,
		},
		Rules: []k8sv1beta1.PolicyRule{
			{
				APIGroups: []string{
					k8sv1beta1.APIGroupAll,
					"extensions",
					"apps",
				},
				Resources: []string{k8sv1beta1.ResourceAll},
				Verbs:     []string{k8sv1beta1.ResourceAll},
			},
		},
	})
	if err != nil && alreadyExistsRegexp.Match([]byte(err.Error())) == false {
		return err
	}

	_, err = kubectlClient.RbacV1beta1().RoleBindings(namespace).Create(&k8sv1beta1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TillerRoleName + "-binding",
			Namespace: namespace,
		},
		Subjects: []k8sv1beta1.Subject{
			{
				Kind:      k8sv1beta1.ServiceAccountKind,
				Name:      TillerServiceAccountName,
				Namespace: tillerNamespace,
			},
		},
		RoleRef: k8sv1beta1.RoleRef{
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
