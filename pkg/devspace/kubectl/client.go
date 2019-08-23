package kubectl

import (
	"fmt"
	"net"
	"net/url"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClient creates a new kubernetes client
func NewClient(devSpaceConfig *latest.Config) (kubernetes.Interface, error) {
	config, err := loadClientConfig(devSpaceConfig, false)
	if err != nil {
		return nil, err
	}

	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(restConfig)
}

// NewClientFromContext creates a new kubernetes client from given context
func NewClientFromContext(context string) (kubernetes.Interface, error) {
	config, err := GetRestConfigFromContext(context)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

// NewClientFromKubeConfig creates a new kubernetes client from a given kube config
func NewClientFromKubeConfig(config clientcmd.ClientConfig) (kubernetes.Interface, error) {
	clientConfig, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(clientConfig)
}

// NewClientWithContextSwitch creates a new kubernetes client and switches the kubectl context
func NewClientWithContextSwitch(devSpaceConfig *latest.Config, switchContext bool) (kubernetes.Interface, error) {
	config, err := loadClientConfig(devSpaceConfig, switchContext)
	if err != nil {
		return nil, err
	}

	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(restConfig)
}

// GetRestConfigBySelect let's the user select a kube context to use
func GetRestConfigBySelect(allowPrivate bool, switchContext bool) (*rest.Config, error) {
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

		if switchContext {
			kubeConfig.CurrentContext = kubeContext
			err = kubeconfig.SaveConfig(kubeConfig)
			if err != nil {
				return nil, errors.Wrap(err, "write kube config")
			}
		}

		return GetRestConfigFromContext(kubeContext)
	}

	return nil, errors.New("We should not reach this point")
}

// GetRestConfigFromContext loads the configuration from a kubernetes context
func GetRestConfigFromContext(context string) (*rest.Config, error) {
	clientConfig, err := kubeconfig.LoadConfigFromContext(context)
	if err != nil {
		return nil, err
	}

	return clientConfig.ClientConfig()
}

// GetRestConfig loads the rest configuration for kubernetes clients and parses it to *rest.Config
func GetRestConfig(config *latest.Config) (*rest.Config, error) {
	clientConfig, err := loadClientConfig(config, false)
	if err != nil {
		return nil, err
	}

	return clientConfig.ClientConfig()
}

func loadClientConfig(config *latest.Config, switchContext bool) (clientcmd.ClientConfig, error) {
	if config == nil {
		return kubeconfig.LoadConfig(), nil
	}

	// Load raw config
	kubeConfig, err := kubeconfig.LoadRawConfig()
	if err != nil {
		return nil, err
	}

	// If we should use a certain kube context use that
	activeContext := kubeConfig.CurrentContext
	if config.Cluster != nil && config.Cluster.KubeContext != nil && len(*config.Cluster.KubeContext) > 0 && activeContext != *config.Cluster.KubeContext {
		activeContext = *config.Cluster.KubeContext

		if switchContext {
			kubeConfig.CurrentContext = activeContext

			err = kubeconfig.SaveConfig(kubeConfig)
			if err != nil {
				return nil, fmt.Errorf("Error saving kube config: %v", err)
			}
		}
	}

	if _, ok := kubeConfig.Contexts[activeContext]; ok == false {
		return nil, fmt.Errorf("Error loading kube config: context '%s' doesn't exist. You may want to run `devspace use space SPACE_NAME`", activeContext)
	}

	// Change context namespace
	if config.Cluster != nil && config.Cluster.Namespace != nil {
		kubeConfig.Contexts[activeContext].Namespace = *config.Cluster.Namespace
	}

	return clientcmd.NewNonInteractiveClientConfig(*kubeConfig, activeContext, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules()), nil
}
