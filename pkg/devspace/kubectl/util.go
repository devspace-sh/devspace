package kubectl

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/util/log"
	"k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ClusterRoleBindingName is the name of the cluster role binding that ensures that the user has enough rights
const ClusterRoleBindingName = "devspace-users"

// EnsureDefaultNamespace makes sure the default namespace exists or will be created
func EnsureDefaultNamespace(client *kubernetes.Clientset, log log.Logger) error {
	config := configutil.GetConfig()
	defaultNamespace, err := configutil.GetDefaultNamespace(config)
	if err != nil {
		return fmt.Errorf("Error getting default namespace: %v", err)
	}

	if defaultNamespace != "default" {
		_, err = client.CoreV1().Namespaces().Get(defaultNamespace, metav1.GetOptions{})
		if err != nil {
			log.Infof("Create namespace %s", defaultNamespace)

			// Create release namespace
			_, err = client.CoreV1().Namespaces().Create(&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: defaultNamespace,
				},
			})
		}
	}

	return err
}

// EnsureGoogleCloudClusterRoleBinding makes sure the needed cluster role is created in the google cloud or a warning is printed
func EnsureGoogleCloudClusterRoleBinding(client *kubernetes.Clientset, log log.Logger) error {
	if IsMinikube() {
		return nil
	}

	_, err := client.RbacV1beta1().ClusterRoleBindings().Get(ClusterRoleBindingName, metav1.GetOptions{})
	if err != nil {
		clusterConfig, _ := GetClientConfig(false)
		if clusterConfig.AuthProvider != nil && clusterConfig.AuthProvider.Name == "gcp" {
			username := configutil.String("")

			log.StartWait("Checking gcloud account")
			gcloudOutput, gcloudErr := exec.Command("gcloud", "config", "list", "account", "--format", "value(core.account)").Output()
			log.StopWait()

			if gcloudErr == nil {
				gcloudEmail := strings.TrimSuffix(strings.TrimSuffix(string(gcloudOutput), "\r\n"), "\n")

				if gcloudEmail != "" {
					username = &gcloudEmail
				}
			}

			if *username == "" {
				return errors.New("Couldn't determine google cloud username. Make sure you are logged in to gcloud")
			}

			rolebinding := &v1beta1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: ClusterRoleBindingName,
				},
				Subjects: []v1beta1.Subject{
					{
						Kind: "User",
						Name: *username,
					},
				},
				RoleRef: v1beta1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "cluster-admin",
				},
			}

			_, err = client.RbacV1beta1().ClusterRoleBindings().Create(rolebinding)
			if err != nil {
				return err
			}
		} else {
			cfg := configutil.GetConfig()
			if cfg.Cluster.CloudProvider == nil || *cfg.Cluster.CloudProvider == "" {
				log.Warn("Unable to check permissions: If you run into errors, please create the ClusterRoleBinding '" + ClusterRoleBindingName + "' as described here: https://devspace.covexo.com/docs/advanced/rbac.html")
			}
		}
	}

	return nil
}
