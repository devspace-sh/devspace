package kubectl

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/portforward"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/pkg/errors"
	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/transport/spdy"
	"net"
	"net/http"
	"os/exec"
	"strings"
)

// ClusterRoleBindingName is the name of the cluster role binding that ensures that the user has enough rights
const ClusterRoleBindingName = "devspace-user"

const minikubeContext = "minikube"
const dockerDesktopContext = "docker-desktop"
const dockerForDesktopContext = "docker-for-desktop"

// WaitStatus are the status to wait
var WaitStatus = []string{
	"ContainerCreating",
	"PodInitializing",
	"Pending",
	"Terminating",
}

// CriticalStatus container status
var CriticalStatus = map[string]bool{
	"Error":                      true,
	"Unknown":                    true,
	"ImagePullBackOff":           true,
	"CrashLoopBackOff":           true,
	"RunContainerError":          true,
	"ErrImagePull":               true,
	"CreateContainerConfigError": true,
	"InvalidImageName":           true,
}

// OkayStatus container status
var OkayStatus = map[string]bool{
	"Completed": true,
	"Running":   true,
}

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
func (client *client) EnsureDeployNamespaces(config *latest.Config, log log.Logger) error {
	namespaces := []string{client.Namespace()}
	for _, imageConfig := range config.Images {
		if imageConfig.Build != nil && imageConfig.Build.Kaniko != nil && imageConfig.Build.Kaniko.Namespace != "" {
			namespaces = append(namespaces, imageConfig.Build.Kaniko.Namespace)
		}
		if imageConfig.Build != nil && imageConfig.Build.BuildKit != nil && imageConfig.Build.BuildKit.InCluster != nil && imageConfig.Build.BuildKit.InCluster.Namespace != "" {
			namespaces = append(namespaces, imageConfig.Build.BuildKit.InCluster.Namespace)
		}
	}
	for _, deployConfig := range config.Deployments {
		if deployConfig.Namespace != "" {
			namespaces = append(namespaces, deployConfig.Namespace)
		}
	}

	for _, namespace := range namespaces {
		_, err := client.KubeClient().CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) == false {
				return errors.Wrap(err, "get namespace")
			}

			// create namespace
			_, err = client.KubeClient().CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}, metav1.CreateOptions{})
			if err != nil && kerrors.IsAlreadyExists(err) == false {
				return err
			}

			log.Donef("Created namespace: %s", namespace)
		}
	}

	return nil
}

// EnsureGoogleCloudClusterRoleBinding makes sure the needed cluster role is created in the google cloud or a warning is printed
func (client *client) EnsureGoogleCloudClusterRoleBinding(log log.Logger) error {
	if client.IsLocalKubernetes() {
		return nil
	}

	_, err := client.KubeClient().RbacV1().ClusterRoleBindings().Get(context.TODO(), ClusterRoleBindingName, metav1.GetOptions{})
	if err != nil {
		if client.restConfig != nil && client.restConfig.AuthProvider != nil && client.restConfig.AuthProvider.Name == "gcp" {
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

			rolebinding := &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: ClusterRoleBindingName,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind: "User",
						Name: *username,
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "cluster-admin",
				},
			}

			_, err = client.KubeClient().RbacV1().ClusterRoleBindings().Create(context.TODO(), rolebinding, metav1.CreateOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// GetPodStatus returns the pod status as a string
// Taken from https://github.com/kubernetes/kubernetes/pkg/printers/internalversion/printers.go
func GetPodStatus(pod *k8sv1.Pod) string {
	reason := string(pod.Status.Phase)

	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	initializing := false

	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]

		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case container.State.Terminated != nil:
			// initialization is failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}
	if !initializing {
		hasRunning := false

		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]

			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else if container.Ready && container.State.Running != nil {
				hasRunning = true
			}
		}

		// change pod status back to "Running" if there is at least one container still reporting as "Running" status
		if reason == "Completed" && hasRunning {
			reason = "Running"
		}
	}

	if pod.DeletionTimestamp != nil && pod.Status.Reason == "NodeLost" {
		reason = "Unknown"
	} else if pod.DeletionTimestamp != nil {
		reason = "Terminating"
	}

	return reason
}

// NewPortForwarder creates a new port forwarder object for the specified pods, ports and addresses
func (client *client) NewPortForwarder(pod *k8sv1.Pod, ports []string, addresses []string, stopChan chan struct{}, readyChan chan struct{}, errorChan chan error) (*portforward.PortForwarder, error) {
	execRequest := client.KubeClient().CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("portforward")

	transport, upgrader, err := client.GetUpgraderWrapper()
	if err != nil {
		return nil, err
	}

	logFile := log.GetFileLogger("portforwarding")
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", execRequest.URL())

	fw, err := portforward.NewOnAddresses(dialer, addresses, ports, stopChan, readyChan, errorChan, logFile, logFile)

	if err != nil {
		return nil, err
	}

	return fw, nil
}

// IsLocalKubernetes returns true if the current context belongs to a local Kubernetes cluster
func (client *client) IsLocalKubernetes() bool {
	return IsLocalKubernetes(client.currentContext)
}

// IsLocalKubernetes returns true if the context belongs to a local Kubernetes cluster
func IsLocalKubernetes(context string) bool {
	return context == minikubeContext || context == dockerDesktopContext || context == dockerForDesktopContext
}
