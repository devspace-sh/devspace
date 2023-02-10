package kubectl

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"net"
	"net/http"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl/portforward"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/transport/spdy"
)

const (
	minikubeContext         = "minikube"
	minikubeProvider        = "minikube.sigs.k8s.io"
	dockerDesktopContext    = "docker-desktop"
	dockerForDesktopContext = "docker-for-desktop"
)

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

// GetPodStatus returns the pod status as a string
// Taken from https://github.com/kubernetes/kubernetes/pkg/printers/internalversion/printers.go
func GetPodStatus(pod *corev1.Pod) string {
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

func EnsureNamespace(ctx context.Context, client Client, namespace string, log log.Logger) error {
	_, err := client.KubeClient().CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			if kerrors.IsForbidden(err) {
				return nil
			}

			return errors.Wrap(err, "get namespace")
		}

		// create namespace
		_, err = client.KubeClient().CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}, metav1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			return err
		}

		log.WithPrefixColor("info ", "cyan+b").Donef("Created namespace: %s", namespace)
	}

	return nil
}

// NewPortForwarder creates a new port forwarder object for the specified pods, ports and addresses
func NewPortForwarder(client Client, pod *corev1.Pod, ports []string, addresses []string, stopChan chan struct{}, readyChan chan struct{}, errorChan chan error) (*portforward.PortForwarder, error) {
	execRequest := client.KubeClient().CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("portforward")

	transport, upgrader, err := GetUpgraderWrapper(client)
	if err != nil {
		return nil, err
	}

	logFile := log.GetFileLogger("portforwarding")
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", execRequest.URL())

	fw, err := portforward.NewOnAddresses(dialer, addresses, ports, stopChan, readyChan, errorChan, logFile.Writer(logrus.InfoLevel, false), logFile.Writer(logrus.WarnLevel, false))
	if err != nil {
		return nil, err
	}

	return fw, nil
}

// IsLocalKubernetes returns true if the context belongs to a local Kubernetes cluster
func IsLocalKubernetes(kubeClient Client) bool {
	if kubeClient == nil {
		return false
	}

	if IsMinikubeKubernetes(kubeClient) {
		return true
	}

	context := kubeClient.CurrentContext()
	if strings.HasPrefix(context, "kind-") ||
		context == dockerDesktopContext ||
		context == dockerForDesktopContext {
		return true
	} else if strings.Contains(context, "vcluster_") &&
		(strings.HasSuffix(context, dockerDesktopContext) ||
			strings.HasSuffix(context, dockerForDesktopContext) ||
			strings.Contains(context, "kind-")) {
		return true
	}

	return false
}

func IsMinikubeKubernetes(kubeClient Client) bool {
	if kubeClient == nil {
		return false
	}

	if kubeClient.CurrentContext() == minikubeContext {
		return true
	}

	if strings.Contains(kubeClient.CurrentContext(), "vcluster_") && strings.HasSuffix(kubeClient.CurrentContext(), minikubeContext) {
		return true
	}

	if kubeClient.ClientConfig() == nil {
		return false
	}

	if rawConfig, err := kubeClient.ClientConfig().RawConfig(); err == nil {
		clusters := rawConfig.Clusters[rawConfig.Contexts[rawConfig.CurrentContext].Cluster]
		for _, extension := range clusters.Extensions {
			ext, err := runtime.DefaultUnstructuredConverter.ToUnstructured(extension)
			if err == nil {
				if provider, ok := ext["provider"].(string); ok {
					if provider == minikubeProvider {
						return true
					}
				}
			}
		}
	}

	return false
}

// GetKindContext returns the kind cluster name
func GetKindContext(context string) string {
	if !strings.HasPrefix(context, "kind-") {
		return ""
	}

	return strings.TrimPrefix(context, "kind-")
}
