package kubectl

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl/minikube"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/mgutz/ansi"
	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kubernetes/pkg/util/node"
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
func EnsureDefaultNamespace(config *latest.Config, client kubernetes.Interface, log log.Logger) error {
	defaultNamespace, err := configutil.GetDefaultNamespace(config)
	if err != nil {
		return fmt.Errorf("Error getting default namespace: %v", err)
	}

	if defaultNamespace != "default" {
		_, err = client.CoreV1().Namespaces().Get(defaultNamespace, metav1.GetOptions{})
		if err != nil {
			// Create release namespace
			_, err = client.CoreV1().Namespaces().Create(&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: defaultNamespace,
				},
			})

			log.Donef("Created namespace %s", defaultNamespace)
		}
	} else {
		log.Warn("You are deploying to the 'default' namespace of your cluster. This is highly discouraged as this namespace cannot be deleted.")
		log.Infof("\r          \nPlease run: \n- `%s` to tell DevSpace to deploy to this namespace \n- `%s` to create a new space in DevSpace Cloud\n- `%s` to use an existing space\n", ansi.Color("devspace use namespace [NAME]", "white+b"), ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"))

		log.StartWait("Will continue to deploy in 5 seconds")
		time.Sleep(5 * time.Second)
		log.StopWait()
	}

	return err
}

// EnsureGoogleCloudClusterRoleBinding makes sure the needed cluster role is created in the google cloud or a warning is printed
func EnsureGoogleCloudClusterRoleBinding(config *latest.Config, client kubernetes.Interface, log log.Logger) error {
	if minikube.IsMinikube(config) {
		return nil
	}

	_, err := client.RbacV1beta1().ClusterRoleBindings().Get(ClusterRoleBindingName, metav1.GetOptions{})
	if err != nil {
		clusterConfig, _ := GetRestConfig(config)
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

// GetNewestRunningPod retrieves the first pod that is found that has the status "Running" using the label selector string
func GetNewestRunningPod(config *latest.Config, kubectl kubernetes.Interface, labelSelector, namespace string, maxWaiting time.Duration) (*k8sv1.Pod, error) {
	if namespace == "" {
		defaultNamespace, err := configutil.GetDefaultNamespace(config)
		if err != nil {
			return nil, err
		}

		namespace = defaultNamespace
	}

	waitingInterval := 1 * time.Second
	for maxWaiting > 0 {
		time.Sleep(waitingInterval)

		podList, err := kubectl.CoreV1().Pods(namespace).List(metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return nil, err
		}

		if podList.Size() > 0 && len(podList.Items) > 0 {
			// Get Pod with latest creation timestamp
			var selectedPod *k8sv1.Pod

			for _, pod := range podList.Items {
				currentPod := pod

				if selectedPod == nil || currentPod.CreationTimestamp.Time.After(selectedPod.CreationTimestamp.Time) {
					selectedPod = &currentPod
				}
			}

			if selectedPod != nil {
				podStatus := GetPodStatus(selectedPod)

				if podStatus == "Running" {
					return selectedPod, nil
				} else if podStatus == "Error" || podStatus == "Unknown" || podStatus == "ImagePullBackOff" || podStatus == "CrashLoopBackOff" || podStatus == "RunContainerError" || podStatus == "ErrImagePull" || podStatus == "CreateContainerConfigError" || podStatus == "InvalidImageName" {
					return nil, fmt.Errorf("Selected Pod(s) cannot start (Status: %s)", podStatus)
				}
			}
		}

		time.Sleep(waitingInterval)
		maxWaiting -= waitingInterval * 2
	}

	return nil, fmt.Errorf("Waiting for pod with selector %s in namespace %s timed out", labelSelector, namespace)
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

	if pod.DeletionTimestamp != nil && pod.Status.Reason == node.NodeUnreachablePodReason {
		reason = "Unknown"
	} else if pod.DeletionTimestamp != nil {
		reason = "Terminating"
	}

	return reason
}

// GetPodsFromDeployment retrieves all found pods from a deployment name
func GetPodsFromDeployment(kubectl kubernetes.Interface, deployment, namespace string) (*k8sv1.PodList, error) {
	deploy, err := kubectl.ExtensionsV1beta1().Deployments(namespace).Get(deployment, metav1.GetOptions{})
	// Deployment not there
	if err != nil {
		return nil, err
	}

	matchLabels := deploy.Spec.Selector.MatchLabels
	if len(matchLabels) <= 0 {
		return nil, errors.New("No matchLabels defined deployment")
	}

	matchLabelString := ""
	for k, v := range matchLabels {
		if len(matchLabelString) > 0 {
			matchLabelString += ","
		}

		matchLabelString += k + "=" + v
	}

	return kubectl.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: matchLabelString,
	})
}

// ForwardPorts forwards the specified ports on the specified interface addresses from the cluster to the local machine
func ForwardPorts(config *latest.Config, kubectlClient kubernetes.Interface, pod *k8sv1.Pod, ports []string, addresses []string, stopChan chan struct{}, readyChan chan struct{}) error {
	fw, err := NewPortForwarder(config, kubectlClient, pod, ports, addresses, stopChan, readyChan)
	if err != nil {
		return err
	}

	return fw.ForwardPorts()
}

// NewPortForwarder creates a new port forwarder object for the specified pods, ports and addresses
func NewPortForwarder(devSpaceConfig *latest.Config, kubectlClient kubernetes.Interface, pod *k8sv1.Pod, ports []string, addresses []string, stopChan chan struct{}, readyChan chan struct{}) (*portforward.PortForwarder, error) {
	config, err := GetRestConfig(devSpaceConfig)
	if err != nil {
		return nil, err
	}

	execRequest := kubectlClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("portforward")

	transport, upgrader, err := GetUpgraderWrapper(config)
	if err != nil {
		return nil, err
	}

	logFile := log.GetFileLogger("portforwarding")
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", execRequest.URL())

	fw, err := portforward.NewOnAddresses(dialer, addresses, ports, stopChan, readyChan, logFile, logFile)

	if err != nil {
		return nil, err
	}

	return fw, nil
}
