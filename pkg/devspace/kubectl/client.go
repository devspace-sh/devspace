package kubectl

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/util"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"io"
	"net"
	"net/url"
	"os"
	"sort"

	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/loft-sh/devspace/pkg/util/terminal"

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
	// CurrentContext returns the current kube context name
	CurrentContext() string

	// KubeClient returns an interface to a kube client
	KubeClient() kubernetes.Interface

	// Namespace returns the default namespace of the kube context
	Namespace() string

	// RestConfig returns the underlying kube rest config
	RestConfig() *rest.Config

	// ClientConfig returns the underlying kube client config
	ClientConfig() clientcmd.ClientConfig

	// KubeConfigLoader returns the kube config loader interface
	KubeConfigLoader() kubeconfig.Loader

	// CopyFromReader copies and extracts files into the container from the reader interface
	CopyFromReader(ctx context.Context, pod *k8sv1.Pod, container, containerPath string, reader io.Reader) error

	// Copy copies and extracts files into the container from the local path excluding the ones specified
	// in the exclude array.
	Copy(ctx context.Context, pod *k8sv1.Pod, container, containerPath, localPath string, exclude []string) error

	// ExecStream starts a new exec request with given options
	ExecStream(ctx context.Context, options *ExecStreamOptions) error

	// ExecBuffered starts a new exec request, waits for it to finish and returns the stdout and stderr to the caller
	ExecBuffered(ctx context.Context, pod *k8sv1.Pod, container string, command []string, input io.Reader) ([]byte, []byte, error)

	// ExecBufferedCombined starts a new exec request, waits for it to finish and returns the output to the caller
	ExecBufferedCombined(ctx context.Context, pod *k8sv1.Pod, container string, command []string, input io.Reader) ([]byte, error)

	// GenericRequest executes a generic kubernetes api request and returns the response as a string
	GenericRequest(ctx context.Context, options *GenericRequestOptions) (string, error)

	// ReadLogs starts a new logs request to the given pod and container
	ReadLogs(ctx context.Context, namespace, podName, containerName string, lastContainerLog bool, tail *int64) (string, error)

	// Logs starts a new logs request to the given pod and container and returns a ReadCloser interface
	// to allow continuous reading. Can also follow a log if specified.
	Logs(ctx context.Context, namespace, podName, containerName string, lastContainerLog bool, tail *int64, follow bool) (io.ReadCloser, error)

	// IsInCluster returns true if in cluster kubernetes configuration is detected
	IsInCluster() bool
}

type client struct {
	Client       kubernetes.Interface
	clientConfig clientcmd.ClientConfig
	restConfig   *rest.Config
	kubeLoader   kubeconfig.Loader

	currentContext string
	namespace      string
	isInCluster    bool
}

var _, tty = terminal.SetupTTY(os.Stdin, os.Stdout)
var isTerminalIn = tty.IsTerminalIn()

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
	restConfig.UserAgent = "DevSpace Version " + upgrade.GetVersion()
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "new client")
	}

	return &client{
		Client:       kubeClient,
		clientConfig: clientConfig,
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
	for {
		kubeContext, err := log.Question(&survey.QuestionOptions{
			Question:     "Which kube context do you want to use",
			DefaultValue: kubeConfig.CurrentContext,
			Options:      options,
		})
		if err != nil {
			return nil, err
		}

		// Check if cluster is in private network
		if !allowPrivate {
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
}

// ClientConfig returns the underlying kube client config
func (client *client) ClientConfig() clientcmd.ClientConfig {
	return client.clientConfig
}

// IsInCluster returns if the kube context is the in cluster context
func (client *client) IsInCluster() bool {
	return client.isInCluster
}

// CheckKubeContext prints a warning if the last kube context is different than this one
func CheckKubeContext(client Client, localCache localcache.Cache, noWarning, autoSwitch bool, log log.Logger) (Client, error) {
	currentConfigContext := &localcache.LastContextConfig{
		Namespace: client.Namespace(),
		Context:   client.CurrentContext(),
	}

	resetClient := false
	if localCache != nil && !noWarning {
		lastConfigContext := localCache.GetLastContext()

		// print warning if context or namespace has changed since last deployment process (expect if explicitly provided as flags)
		if lastConfigContext != nil {
			// if the current kubeContext!=last kubeContext
			// then ask which kubeContext to use
			// else if the current namespace!=last namespace
			// then ask which namespace to use
			if lastConfigContext.Context != "" &&
				lastConfigContext.Context != currentConfigContext.Context {
				if autoSwitch {
					currentConfigContext.Context = lastConfigContext.Context
					currentConfigContext.Namespace = lastConfigContext.Namespace
					resetClient = true
				} else if log.GetLevel() >= logrus.InfoLevel {
					log.WriteString(logrus.WarnLevel, "\n")
					log.Warnf(ansi.Color("Are you using the correct kube context?", "white+b"))
					log.Warnf("Current kube context: '%s'", ansi.Color(currentConfigContext.Context, "white+b"))
					log.Warnf("Last    kube context: '%s'", ansi.Color(lastConfigContext.Context, "white+b"))

					// if terminal is not interactive then return the same client
					if !isTerminalIn {
						return client, nil
					}

					kc, err := log.Question(&survey.QuestionOptions{
						Question:     "Which context do you want to use?",
						DefaultValue: currentConfigContext.Context,
						Options: []string{
							currentConfigContext.Context,
							lastConfigContext.Context,
						},
					})
					if err != nil {
						return client, err
					}
					if kc != currentConfigContext.Context {
						currentConfigContext.Context = kc
						currentConfigContext.Namespace = lastConfigContext.Namespace
						resetClient = true
					}
				}
			} else if lastConfigContext.Namespace != "" &&
				lastConfigContext.Namespace != currentConfigContext.Namespace {
				if autoSwitch {
					currentConfigContext.Namespace = lastConfigContext.Namespace
					resetClient = true
				} else if log.GetLevel() >= logrus.InfoLevel {
					log.WriteString(logrus.WarnLevel, "\n")
					log.Warnf(ansi.Color("Are you using the correct namespace?", "white+b"))
					log.Warnf("Current namespace: '%s'", ansi.Color(currentConfigContext.Namespace, "white+b"))
					log.Warnf("Last    namespace: '%s'", ansi.Color(lastConfigContext.Namespace, "white+b"))

					// if terminal is not interactive then return the same client
					if !isTerminalIn {
						return client, nil
					}

					ns, err := log.Question(&survey.QuestionOptions{
						Question:     "Which namespace do you want to use?",
						DefaultValue: currentConfigContext.Namespace,
						Options: []string{
							currentConfigContext.Namespace,
							lastConfigContext.Namespace,
						},
					})
					if err != nil {
						return client, err
					}
					if ns != currentConfigContext.Namespace {
						currentConfigContext.Namespace = ns
						resetClient = true
					}
				}
			}
		}

		// Warn if using default namespace unless previous deployment was also to default namespace
		if isTerminalIn &&
			log.GetLevel() >= logrus.InfoLevel &&
			currentConfigContext.Namespace == metav1.NamespaceDefault &&
			(lastConfigContext == nil || lastConfigContext.Namespace != metav1.NamespaceDefault) {
			log.Warn("Deploying into the 'default' namespace is usually not a good idea as this namespace cannot be deleted")
			log.Warn("Please use 'devspace use namespace my-namespace' to select a different one\n")
			useDefault, err := log.Question(&survey.QuestionOptions{
				Question:     "Are you sure you want to use the 'default' namespace?",
				DefaultValue: "No",
				Options: []string{
					"No",
					"Yes",
				},
			})
			if err != nil {
				return client, err
			} else if useDefault == "No" {
				return nil, fmt.Errorf("please run 'devspace use namespace my-namespace' to select a different namespace before rerunning")
			}
		}
	}

	// Save changes to cache
	if localCache != nil {
		// Save changes to cache
		localCache.SetLastContext(&localcache.LastContextConfig{
			Context:   currentConfigContext.Context,
			Namespace: currentConfigContext.Namespace,
		})
		err := localCache.Save()
		if err != nil {
			log.Warnf("Error saving cache: %v", err)
		}
	}

	// Info messages
	log.Infof("Using namespace '%s'", ansi.Color(currentConfigContext.Namespace, "white+b"))
	log.Infof("Using kube context '%s'", ansi.Color(currentConfigContext.Context, "white+b"))
	if resetClient {
		return NewClientFromContext(currentConfigContext.Context, currentConfigContext.Namespace, true, client.KubeConfigLoader())
	}

	return client, nil
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
