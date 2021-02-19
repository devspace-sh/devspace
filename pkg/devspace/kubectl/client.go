package kubectl

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/util"

	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/portforward"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/survey"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client holds all kubernetes related functions
type Client interface {
	// Returns the current kube context name
	CurrentContext() string

	// Returns an interface to a kube client
	KubeClient() kubernetes.Interface

	// Returns the default namespace of the kube context
	Namespace() string

	// Returns the underlying kube rest config
	RestConfig() *rest.Config

	// Returns the kube config loader interface
	KubeConfigLoader() kubeconfig.Loader

	// This function will print a warning if the generated config contains a different last kube context / namespace
	// than the one that is used currently
	PrintWarning(generatedConfig *generated.Config, noWarning, shouldWait bool, log log.Logger) error

	// Copies and extracts files into the container from the reader interface
	CopyFromReader(pod *k8sv1.Pod, container, containerPath string, reader io.Reader) error

	// Copies and extracts files into the container from the local path excluding the ones specified
	// in the exclude array.
	Copy(pod *k8sv1.Pod, container, containerPath, localPath string, exclude []string) error

	// Starts a new exec request with given options and custom transport
	ExecStreamWithTransport(options *ExecStreamWithTransportOptions) error

	// Starts a new exec request with given options
	ExecStream(options *ExecStreamOptions) error

	// Starts a new exec request, waits for it to finish and returns the stdout and stderr to the caller
	ExecBuffered(pod *k8sv1.Pod, container string, command []string, input io.Reader) ([]byte, []byte, error)

	// Executes a generic kubernetes api request and returns the response as a string
	GenericRequest(options *GenericRequestOptions) (string, error)

	// Starts a new logs request to the given pod and container
	ReadLogs(namespace, podName, containerName string, lastContainerLog bool, tail *int64) (string, error)

	// Starts a new logs request to the given pod and container and returns a ReadCloser interface
	// to allow continous reading. Can also follow a log if specified.
	Logs(ctx context.Context, namespace, podName, containerName string, lastContainerLog bool, tail *int64, follow bool) (io.ReadCloser, error)

	// Creates a new round tripper and upgrade wrapper for the current kube config
	GetUpgraderWrapper() (http.RoundTripper, UpgraderWrapper, error)

	// Ensures the config names exist and if not creates them
	EnsureDeployNamespaces(config *latest.Config, log log.Logger) error

	// Ensures a google cloud cluster role binding is created in GKE like clusters
	EnsureGoogleCloudClusterRoleBinding(log log.Logger) error

	// Creates a new port forwarder object for the current kube context to the given pod
	NewPortForwarder(pod *k8sv1.Pod, ports []string, addresses []string, stopChan chan struct{}, readyChan chan struct{}, errorChan chan error) (*portforward.PortForwarder, error)

	// Returns true if a local kubernetes installation such as minikube is detected
	IsLocalKubernetes() bool

	// Returns true if in cluster kubernetes configuration is detected
	IsInCluster() bool
}

type client struct {
	Client       kubernetes.Interface
	ClientConfig clientcmd.ClientConfig
	restConfig   *rest.Config
	kubeLoader   kubeconfig.Loader

	currentContext string
	namespace      string
	isInCluster    bool
}

// NewDefaultClient creates the new default kube client from the active context @Factory
func NewDefaultClient() (Client, error) {
	return NewClientFromContext("", "", false, kubeconfig.NewLoader())
}

// NewClientFromContext creates a new kubernetes client from given context @Factory
func NewClientFromContext(context, namespace string, switchContext bool, kubeLoader kubeconfig.Loader) (Client, error) {
	clientConfig, activeContext, activeNamespace, isInCluster, err := util.NewClientByContext(context, namespace, switchContext, kubeLoader)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "new client")
	}

	return &client{
		Client:       kubeClient,
		ClientConfig: clientConfig,
		restConfig:   restConfig,
		kubeLoader:   kubeLoader,

		namespace:      activeNamespace,
		currentContext: activeContext,
		isInCluster:    isInCluster,
	}, nil
}

// NewClientBySelect creates a new kubernetes client by user select @Factory
func NewClientBySelect(allowPrivate bool, switchContext bool, kubeLoader kubeconfig.Loader, log log.Logger) (Client, error) {
	kubeConfig, err := kubeLoader.LoadRawConfig()
	if err != nil {
		return nil, err
	}

	// Get all kube contexts
	options := make([]string, 0, len(kubeConfig.Contexts))
	for context := range kubeConfig.Contexts {
		options = append(options, context)
	}
	if len(options) == 0 {
		return nil, errors.New("No kubectl context found. Make sure kubectl is installed and you have a working kubernetes context configured")
	}

	sort.Strings(options)
	for true {
		kubeContext, err := log.Question(&survey.QuestionOptions{
			Question:     "Which kube context do you want to use",
			DefaultValue: kubeConfig.CurrentContext,
			Options:      options,
		})
		if err != nil {
			return nil, err
		}

		// Check if cluster is in private network
		if allowPrivate == false {
			context := kubeConfig.Contexts[kubeContext]
			cluster := kubeConfig.Clusters[context.Cluster]

			url, err := url.Parse(cluster.Server)
			if err != nil {
				return nil, errors.Wrap(err, "url parse")
			}

			ip := net.ParseIP(url.Hostname())
			if ip != nil {
				if IsPrivateIP(ip) {
					log.Infof("Clusters with private ips (%s) cannot be used", url.Hostname())
					continue
				}
			}
		}

		return NewClientFromContext(kubeContext, "", switchContext, kubeLoader)
	}

	return nil, errors.New("We should not reach this point")
}

// IsInCluster returns if the kube context is the in cluster context
func (client *client) IsInCluster() bool {
	return client.isInCluster
}

// PrintWarning prints a warning if the last kube context is different than this one
func (client *client) PrintWarning(generatedConfig *generated.Config, noWarning, shouldWait bool, log log.Logger) error {
	if generatedConfig != nil && log.GetLevel() >= logrus.InfoLevel && noWarning == false {
		// print warning if context or namespace has changed since last deployment process (expect if explicitly provided as flags)
		if generatedConfig.GetActive().LastContext != nil {
			wait := false

			if generatedConfig.GetActive().LastContext.Context != "" && generatedConfig.GetActive().LastContext.Context != client.currentContext {
				log.WriteString("\n")
				log.Warnf(ansi.Color("Are you using the correct kube context?", "white+b"))
				log.Warnf("Current kube context: '%s'", ansi.Color(client.currentContext, "white+b"))
				log.Warnf("Last    kube context: '%s'", ansi.Color(generatedConfig.GetActive().LastContext.Context, "white+b"))
				log.WriteString("\n")

				log.Infof("Use the '%s' flag to switch to the context and namespace previously used to deploy this project", ansi.Color("-s / --switch-context", "white+b"))
				log.Infof("Or use the '%s' flag to ignore this warning", ansi.Color("--no-warn", "white+b"))
				wait = true
			} else if generatedConfig.GetActive().LastContext.Namespace != "" && generatedConfig.GetActive().LastContext.Namespace != client.namespace {
				log.WriteString("\n")
				log.Warnf(ansi.Color("Are you using the correct namespace?", "white+b"))
				log.Warnf("Current namespace: '%s'", ansi.Color(client.namespace, "white+b"))
				log.Warnf("Last    namespace: '%s'", ansi.Color(generatedConfig.GetActive().LastContext.Namespace, "white+b"))
				log.WriteString("\n")

				log.Infof("Use the '%s' flag to switch to the context and namespace previously used to deploy this project", ansi.Color("-s / --switch-context", "white+b"))
				log.Infof("Or use the '%s' flag to ignore this warning", ansi.Color("--no-warn", "white+b"))
				wait = true
			}

			if wait && shouldWait {
				log.StartWait("Will continue in 10 seconds...")
				time.Sleep(10 * time.Second)
				log.StopWait()
				log.WriteString("\n")
			}
		}

		// Warn if using default namespace unless previous deployment was also to default namespace
		if shouldWait && client.namespace == metav1.NamespaceDefault && (generatedConfig.GetActive().LastContext == nil || generatedConfig.GetActive().LastContext.Namespace != metav1.NamespaceDefault) {
			log.Warn("Deploying into the 'default' namespace is usually not a good idea as this namespace cannot be deleted\n")
			log.StartWait("Will continue in 5 seconds...")
			time.Sleep(5 * time.Second)
			log.StopWait()
		}
	}

	// Info messages
	log.Infof("Using namespace '%s'", ansi.Color(client.namespace, "white+b"))
	log.Infof("Using kube context '%s'", ansi.Color(client.currentContext, "white+b"))

	return nil
}

func (client *client) CurrentContext() string {
	return client.currentContext
}

func (client *client) KubeClient() kubernetes.Interface {
	return client.Client
}

func (client *client) Namespace() string {
	return client.namespace
}

func (client *client) RestConfig() *rest.Config {
	return client.restConfig
}

func (client *client) KubeConfigLoader() kubeconfig.Loader {
	return client.kubeLoader
}
