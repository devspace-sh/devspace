package kubectl

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client holds all important information for kubernetes
type Client struct {
	Client       kubernetes.Interface
	ClientConfig clientcmd.ClientConfig
	RestConfig   *rest.Config

	CurrentContext string
	Namespace      string
}

// UpdateLastContext returns a new kubectl client based on the namespace flag and kube context flag and updates the generated accordingly
func (client *Client) UpdateLastContext(updateGenerated bool, log log.Logger) (*Client, error) {
	// Info messages
	log.Infof("Using kube context '%s'", ansi.Color(client.CurrentContext, "white+b"))
	log.Infof("Using namespace '%s'", ansi.Color(client.Namespace, "white+b"))

	generatedConfig, err := generated.LoadConfig()
	if err == nil {
		// print warning if context or namespace has changed since last deployment process (expect if explicitly provided as flags)
		if generatedConfig.LastContext != nil {
			if (generatedConfig.LastContext.Context != "" && generatedConfig.LastContext.Context != client.CurrentContext) || (generatedConfig.LastContext.Namespace != "" && generatedConfig.LastContext.Namespace != client.Namespace) {
				log.WriteString("\n")
				log.Warnf("Your current kube-context and/or default namespace is different than last time.")
				log.WriteString("\n")

				if updateGenerated {
					log.Warn(ansi.Color("Abort with CTRL+C if you are using the wrong kube-context.", "red+b"))
					log.StartWait("Will continue in 10 seconds...")
					time.Sleep(10 * time.Second)
					log.StopWait()
					log.WriteString("\n")
				}
			}
		}

		// warn if user is currently using default namespace but only if we updating the generated config, since we don't want the warning in devspace enter, logs etc.
		if updateGenerated && client.Namespace == metav1.NamespaceDefault {
			log.Warn("Using the 'default' namespace of your cluster is highly discouraged as this namespace cannot be deleted.")

			log.Warn(ansi.Color("Abort with CTRL+C if you do not want to use the default namespace.", "red+b"))
			log.StartWait("Will continue in 5 seconds...")
			time.Sleep(5 * time.Second)
			log.StopWait()
			log.WriteString("\n")
		}

		// Update generated if we deploy the application
		if updateGenerated {
			generatedConfig.LastContext = &generated.LastContextConfig{
				Context:   client.CurrentContext,
				Namespace: client.Namespace,
			}

			err = generated.SaveConfig(generatedConfig)
			if err != nil {
				return nil, errors.Wrap(err, "save generated")
			}
		}
	}

	return client, nil
}

// NewClientFromContext creates a new kubernetes client from given context
func NewClientFromContext(context, namespace string, switchContext bool) (*Client, error) {
	// Load raw config
	kubeConfig, err := kubeconfig.LoadRawConfig()
	if err != nil {
		return nil, err
	}

	// If we should use a certain kube context use that
	activeContext := kubeConfig.CurrentContext
	if context != "" && activeContext != context {
		activeContext = context
		if switchContext {
			kubeConfig.CurrentContext = activeContext

			err = kubeconfig.SaveConfig(kubeConfig)
			if err != nil {
				return nil, fmt.Errorf("Error saving kube config: %v", err)
			}
		}
	}

	if _, ok := kubeConfig.Contexts[activeContext]; ok == false {
		return nil, fmt.Errorf("Error loading kube config, context '%s' doesn't exist", activeContext)
	}

	// Change context namespace
	activeNamespace := kubeConfig.Contexts[activeContext].Namespace
	if namespace != "" && activeNamespace != namespace {
		activeNamespace = namespace
		kubeConfig.Contexts[activeContext].Namespace = namespace
	}
	if activeNamespace == "" {
		activeNamespace = metav1.NamespaceDefault
	}

	clientConfig := clientcmd.NewNonInteractiveClientConfig(*kubeConfig, activeContext, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules())

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &Client{
		Client:       client,
		ClientConfig: clientConfig,
		RestConfig:   restConfig,

		Namespace:      activeNamespace,
		CurrentContext: activeContext,
	}, nil
}

// NewClientBySelect creates a new kubernetes client by user select
func NewClientBySelect(allowPrivate bool, switchContext bool) (*Client, error) {
	kubeConfig, err := kubeconfig.LoadRawConfig()
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

	for true {
		kubeContext := survey.Question(&survey.QuestionOptions{
			Question:     "Which kube context do you want to use",
			DefaultValue: kubeConfig.CurrentContext,
			Options:      options,
		})

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

		return NewClientFromContext(kubeContext, "", switchContext)
	}

	return nil, errors.New("We should not reach this point")
}
