package kubectl

import (
	"github.com/covexo/devspace/pkg/util/terminal"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/covexo/devspace/pkg/util/kubeconfig"

	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/log"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	k8sapi "k8s.io/kubernetes/pkg/apis/core"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/kubectl/util/term"
	"k8s.io/kubernetes/pkg/printers"
	describe "k8s.io/kubernetes/pkg/printers/internalversion"
	"k8s.io/kubernetes/pkg/util/node"
)

var isMinikubeVar *bool
var loadCloudConfigOnce sync.Once

//NewClient creates a new kubernetes client
func NewClient() (*kubernetes.Clientset, error) {
	config, err := GetClientConfig(false)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

// NewClientWithContextSwitch creates a new kubernetes client and switches the kubectl context
func NewClientWithContextSwitch(switchContext bool) (*kubernetes.Clientset, error) {
	config, err := GetClientConfig(switchContext)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

//GetClientConfig loads the configuration for kubernetes clients and parses it to *rest.Config
func GetClientConfig(switchContext bool) (*rest.Config, error) {
	var err error

	config := configutil.GetConfig()
	if config.Cluster == nil {
		return nil, errors.New("Couldn't load cluster config, did you run devspace init")
	}

	// Update devspace cloud cluster config
	if config.Cluster.CloudProvider != nil && *config.Cluster.CloudProvider != "" {
		loadCloudConfigOnce.Do(func() {
			err = loadCloudConfig(config, log.GetInstance())
		})
		if err != nil {
			return nil, err
		}
	}

	// Use kube config if desired
	if config.Cluster.APIServer == nil {
		kubeConfig, err := kubeconfig.ReadKubeConfig(clientcmd.RecommendedHomeFile)
		if err != nil {
			return nil, err
		}

		activeContext := kubeConfig.CurrentContext

		// If we should use a certain kube context use that
		if config.Cluster != nil && config.Cluster.KubeContext != nil && len(*config.Cluster.KubeContext) > 0 && activeContext != *config.Cluster.KubeContext {
			activeContext = *config.Cluster.KubeContext

			if switchContext {
				kubeConfig.CurrentContext = activeContext

				err = kubeconfig.WriteKubeConfig(kubeConfig, clientcmd.RecommendedHomeFile)
				if err != nil {
					return nil, fmt.Errorf("Error saving kube config: %v", err)
				}
			}
		}

		// Change context namespace
		if config.Cluster != nil && config.Cluster.Namespace != nil {
			kubeConfig.Contexts[activeContext].Namespace = *config.Cluster.Namespace
		}

		return clientcmd.NewNonInteractiveClientConfig(*kubeConfig, activeContext, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules()).ClientConfig()
	}

	// We create a new config object here
	kubeAuthInfo := api.NewAuthInfo()
	if config.Cluster.User != nil {
		if config.Cluster.User.ClientCert != nil {
			kubeAuthInfo.ClientCertificateData = []byte(*config.Cluster.User.ClientCert)
		}
		if config.Cluster.User.ClientKey != nil {
			kubeAuthInfo.ClientKeyData = []byte(*config.Cluster.User.ClientKey)
		}
		if config.Cluster.User.Token != nil {
			kubeAuthInfo.Token = *config.Cluster.User.Token
		}
	}

	kubeCluster := api.NewCluster()
	if config.Cluster.APIServer != nil {
		kubeCluster.Server = *config.Cluster.APIServer
	}
	if config.Cluster.CaCert != nil {
		kubeCluster.CertificateAuthorityData = []byte(*config.Cluster.CaCert)
	}

	kubeContext := api.NewContext()
	kubeContext.Cluster = "devspace"
	kubeContext.AuthInfo = "devspace"

	// Change context namespace
	if config.Cluster.Namespace != nil {
		kubeContext.Namespace = *config.Cluster.Namespace
	}

	kubeConfig := api.NewConfig()
	kubeConfig.AuthInfos["devspace"] = kubeAuthInfo
	kubeConfig.Clusters["devspace"] = kubeCluster
	kubeConfig.Contexts["devspace"] = kubeContext
	kubeConfig.CurrentContext = "devspace"

	return clientcmd.NewNonInteractiveClientConfig(*kubeConfig, "devspace", &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules()).ClientConfig()
}

func loadCloudConfig(config *v1.Config, log log.Logger) error {
	providerConfig, err := cloud.ParseCloudConfig()
	if err != nil {
		return fmt.Errorf("Couldn't load cloud provider config: %v", err)
	}

	log.StartWait("Login to cloud provider")
	err = cloud.Update(providerConfig, *config.Cluster.CloudProvider, config, config.Cluster.APIServer == nil, false)
	log.StopWait()
	if err != nil {
		log.Warnf("Couldn't update cloud provider %s information: %v", *config.Cluster.CloudProvider, err)
	}

	err = configutil.SaveConfig()
	if err != nil {
		return fmt.Errorf("Error saving config: %v", err)
	}

	return nil
}

// IsMinikube returns true if the Kubernetes cluster is a minikube
func IsMinikube() bool {
	if isMinikubeVar == nil {
		isMinikube := false
		config := configutil.GetConfig()
		if config.Cluster.APIServer == nil {
			if config.Cluster.KubeContext == nil {
				loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
				kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
				cfg, err := kubeConfig.RawConfig()
				if err != nil {
					return false
				}

				isMinikube = cfg.CurrentContext == "minikube"
			} else {
				isMinikube = *config.Cluster.KubeContext == "minikube"
			}
		}

		isMinikubeVar = &isMinikube
	}

	return *isMinikubeVar
}

// GetNewestRunningPod retrieves the first pod that is found that has the status "Running" using the label selector string
func GetNewestRunningPod(kubectl *kubernetes.Clientset, labelSelector, namespace string) (*k8sv1.Pod, error) {
	config := configutil.GetConfig()

	if namespace == "" {
		defaultNamespace, err := configutil.GetDefaultNamespace(config)
		if err != nil {
			return nil, err
		}

		namespace = defaultNamespace
	}

	maxWaiting := 120 * time.Second
	waitingInterval := 1 * time.Second

	for maxWaiting > 0 {
		time.Sleep(waitingInterval)

		podList, err := kubectl.Core().Pods(namespace).List(metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			log.Info("Error here")

			return nil, err
		}

		if podList.Size() > 0 && len(podList.Items) > 0 {
			// Get Pod with latest creation timestamp
			var selectedPod *k8sv1.Pod

			for _, pod := range podList.Items {
				if selectedPod == nil || pod.CreationTimestamp.Time.After(selectedPod.CreationTimestamp.Time) {
					selectedPod = &pod
				}
			}

			if selectedPod != nil && GetPodStatus(selectedPod) == "Running" {
				return selectedPod, nil
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

// DescribePod returns a desription string of a pod (internally calls the kubectl describe function)
func DescribePod(namespace, name string) (string, error) {
	newConfig, err := GetClientConfig(false)

	if err != nil {
		return "", err
	}

	newKubectl, err := clientset.NewForConfig(newConfig)

	if err != nil {
		return "", err
	}

	podDescriber := &describe.PodDescriber{newKubectl}

	return podDescriber.Describe(namespace, name, printers.DescriberSettings{ShowEvents: true})
}

// GetPodsFromDeployment retrieves all found pods from a deployment name
func GetPodsFromDeployment(kubectl *kubernetes.Clientset, deployment, namespace string) (*k8sv1.PodList, error) {
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

	return kubectl.Core().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: matchLabelString,
	})
}

// ForwardPorts forwards the specified ports from the cluster to the local machine
func ForwardPorts(kubectlClient *kubernetes.Clientset, pod *k8sv1.Pod, ports []string, stopChan chan struct{}, readyChan chan struct{}) error {
	config, err := GetClientConfig(false)
	if err != nil {
		return err
	}

	execRequest := kubectlClient.Core().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return err
	}

	logFile := log.GetFileLogger("portforwarding")
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", execRequest.URL())
	fw, err := portforward.New(dialer, ports, stopChan, readyChan, logFile, logFile)

	if err != nil {
		return err
	}

	return fw.ForwardPorts()
}

//Exec executes a command for kubectl
func Exec(kubectlClient *kubernetes.Clientset, pod *k8sv1.Pod, container string, command []string, tty bool, errorChannel chan<- error) (io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
	var t term.TTY

	kubeconfig, err := GetClientConfig(false)
	if err != nil {
		return nil, nil, nil, err
	}

	execRequest := kubectlClient.Core().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")

	if tty {
		t = terminal.SetupTTY()
	}

	execRequest.VersionedParams(&k8sapi.PodExecOptions{
		Container: container,
		Command:   command,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       t.Raw,
	}, legacyscheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(kubeconfig, "POST", execRequest.URL())

	if err != nil {
		return nil, nil, nil, err
	}

	if tty {
		var sizeQueue remotecommand.TerminalSizeQueue

		if t.Raw {
			// this call spawns a goroutine to monitor/update the terminal size
			sizeQueue = t.MonitorSize(t.GetSize())
		}

		fn := func() error {
			return exec.Stream(remotecommand.StreamOptions{
				Stdin:             os.Stdin,
				Stdout:            os.Stdout,
				Stderr:            os.Stderr,
				Tty:               t.Raw,
				TerminalSizeQueue: sizeQueue,
			})
		}

		if err := t.Safe(fn); err != nil {
			return nil, nil, nil, err
		}

		return nil, nil, nil, nil
	}
	stdinReader, stdinWriter, _ := os.Pipe()
	stdoutReader, stdoutWriter, _ := os.Pipe()
	stderrReader, stderrWriter, _ := os.Pipe()

	go func() {
		streamErr := exec.Stream(remotecommand.StreamOptions{
			Stdin:  stdinReader,
			Stdout: stdoutWriter,
			Stderr: stderrWriter,
			Tty:    tty,
		})
		stdinWriter.Close()
		stdoutWriter.Close()
		stderrWriter.Close()

		errorChannel <- streamErr
	}()
	return stdinWriter, stdoutReader, stderrReader, nil
}

//ExecBuffered executes a command for kubernetes and returns the output and error buffers
func ExecBuffered(kubectlClient *kubernetes.Clientset, pod *k8sv1.Pod, container string, command []string) ([]byte, []byte, error) {
	_, stdout, stderr, execErr := Exec(kubectlClient, pod, container, command, false, nil)

	if execErr != nil {
		return nil, nil, execErr
	}
	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}

	streamDone := &sync.WaitGroup{}
	streamDone.Add(2)

	go func() {
		io.Copy(stdoutBuffer, stdout)
		streamDone.Done()
	}()

	go func() {
		io.Copy(stderrBuffer, stderr)
		streamDone.Done()
	}()
	streamDone.Wait()

	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), nil
}
