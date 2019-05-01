package kubectl

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl/minikube"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ClusterRoleBindingName is the name of the cluster role binding that ensures that the user has enough rights
const ClusterRoleBindingName = "devspace-user"

var privateIPBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

// IsPrivateIP checks if a given ip is private
func IsPrivateIP(ip net.IP) bool {
	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}

// EnsureDefaultNamespace makes sure the default namespace exists or will be created
func EnsureDefaultNamespace(client kubernetes.Interface, log log.Logger) error {
	config := configutil.GetConfig()
	defaultNamespace, err := configutil.GetDefaultNamespace(config)
	if err != nil {
		return fmt.Errorf("Error getting default namespace: %v", err)
	}

	if defaultNamespace != "default" {
		_, err = client.CoreV1().Namespaces().Get(defaultNamespace, metav1.GetOptions{})
		if err != nil {
			log.Donef("Create namespace %s", defaultNamespace)

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
func EnsureGoogleCloudClusterRoleBinding(client kubernetes.Interface, log log.Logger) error {
	if minikube.IsMinikube() {
		return nil
	}

	_, err := client.RbacV1beta1().ClusterRoleBindings().Get(ClusterRoleBindingName, metav1.GetOptions{})
	if err != nil {
		clusterConfig, _ := GetClientConfig()
		if clusterConfig.AuthProvider != nil && clusterConfig.AuthProvider.Name == "gcp" {
			username := ptr.String("")

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
		}
	}

	return nil
}
