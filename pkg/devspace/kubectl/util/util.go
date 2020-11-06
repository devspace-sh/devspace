package util

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const localContext = "local"

func NewClientByContext(context, namespace string, switchContext bool, kubeLoader kubeconfig.Loader) (clientcmd.ClientConfig, string, string, error) {
	// Load new raw config
	kubeConfigOriginal, err := kubeLoader.LoadRawConfig()
	if err != nil {
		return nil, "", "", err
	}

	// We clone the config here to avoid changing the single loaded config
	kubeConfig := clientcmdapi.Config{}
	err = util.Convert(&kubeConfigOriginal, &kubeConfig)
	if err != nil {
		return nil, "", "", err
	}

	if len(kubeConfig.Clusters) == 0 {
		// try to load in cluster config
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, "", "", errors.Errorf("kube config is invalid: please make sure you have an existing valid kube config")
		}

		rawConfig, err := ConvertRestConfigToRawConfig(config)
		if err != nil {
			return nil, "", "", errors.Wrap(err, "convert in cluster config")
		}

		return clientcmd.NewNonInteractiveClientConfig(*rawConfig, localContext, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules()), localContext, "default", nil
	}

	// If we should use a certain kube context use that
	var (
		activeContext   = kubeConfig.CurrentContext
		activeNamespace = metav1.NamespaceDefault
		saveConfig      = false
	)

	// Set active context
	if context != "" && activeContext != context {
		activeContext = context
		if switchContext {
			kubeConfig.CurrentContext = activeContext
			saveConfig = true
		}
	}

	// Set active namespace
	if kubeConfig.Contexts[activeContext] != nil {
		if kubeConfig.Contexts[activeContext].Namespace != "" {
			activeNamespace = kubeConfig.Contexts[activeContext].Namespace
		}

		if namespace != "" && activeNamespace != namespace {
			activeNamespace = namespace
			kubeConfig.Contexts[activeContext].Namespace = activeNamespace
			if switchContext {
				saveConfig = true
			}
		}
	}

	// Should we save the kube config?
	if saveConfig {
		err = kubeLoader.SaveConfig(&kubeConfig)
		if err != nil {
			return nil, "", "", errors.Errorf("Error saving kube config: %v", err)
		}
	}

	clientConfig := clientcmd.NewNonInteractiveClientConfig(kubeConfig, activeContext, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules())
	if kubeConfig.Contexts[activeContext] == nil {
		return nil, "", "", errors.Errorf("Error loading kube config, context '%s' doesn't exist", activeContext)
	}

	return clientConfig, activeContext, activeNamespace, nil
}

func ConvertRestConfigToRawConfig(config *rest.Config) (*clientcmdapi.Config, error) {
	contextName := localContext
	kubeConfig := clientcmdapi.NewConfig()
	kubeConfig.Contexts = map[string]*clientcmdapi.Context{
		contextName: {
			Cluster:  contextName,
			AuthInfo: contextName,
		},
	}
	kubeConfig.Clusters = map[string]*clientcmdapi.Cluster{
		contextName: {
			Server:                   config.Host,
			InsecureSkipTLSVerify:    config.Insecure,
			CertificateAuthorityData: config.CAData,
			CertificateAuthority:     config.CAFile,
		},
	}
	kubeConfig.AuthInfos = map[string]*clientcmdapi.AuthInfo{
		contextName: {
			Token:                 config.BearerToken,
			TokenFile:             config.BearerTokenFile,
			Impersonate:           config.Impersonate.UserName,
			ImpersonateGroups:     config.Impersonate.Groups,
			ImpersonateUserExtra:  config.Impersonate.Extra,
			ClientCertificate:     config.CertFile,
			ClientCertificateData: config.CertData,
			ClientKey:             config.KeyFile,
			ClientKeyData:         config.KeyData,
			Username:              config.Username,
			Password:              config.Password,
			AuthProvider:          config.AuthProvider,
			Exec:                  config.ExecProvider,
		},
	}
	kubeConfig.CurrentContext = contextName
	raw, err := clientcmd.NewDefaultClientConfig(*kubeConfig, &clientcmd.ConfigOverrides{}).RawConfig()
	return &raw, err
}
