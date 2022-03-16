package kubeconfig

import (
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// AuthCommand is the name of the command used to get auth token for kube-context of Spaces
const AuthCommand = "devspace"

// NewConfig loads a new kube config
func (l *loader) NewConfig() clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
}

// LoadConfig loads the kube config with the default loading rules
func (l *loader) LoadConfig() clientcmd.ClientConfig {
	return l.NewConfig()
}

// LoadRawConfig loads the raw kube config with the default loading rules
func (l *loader) LoadRawConfig() (*api.Config, error) {
	config, err := l.LoadConfig().RawConfig()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// GetCurrentContext retrieves the current kube context
func (l *loader) GetCurrentContext() (string, error) {
	config, err := l.LoadRawConfig()
	if err != nil {
		return "", err
	}

	return config.CurrentContext, nil
}

// SaveConfig writes the kube config back to the specified filename
func (l *loader) SaveConfig(config *api.Config) error {
	err := clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), *config, false)
	if err != nil {
		return err
	}

	return nil
}

// DeleteKubeContext removes the specified devspace id from the kube context if it exists
func (l *loader) DeleteKubeContext(kubeConfig *api.Config, kubeContext string) error {
	// Get context
	contextRaw, ok := kubeConfig.Contexts[kubeContext]
	if !ok {
		// return errors.Errorf("Unable to find current kube-context '%s' in kube-config file", kubeContext)
		// This is debatable but usually we don't care when the context is not there
		return nil
	}

	// Remove context
	delete(kubeConfig.Contexts, kubeContext)

	removeAuthInfo := true
	removeCluster := true

	// Check if AuthInfo or Cluster is used by any other context
	for name, ctx := range kubeConfig.Contexts {
		if name != kubeContext && ctx.AuthInfo == contextRaw.AuthInfo {
			removeAuthInfo = false
		}

		if name != kubeContext && ctx.Cluster == contextRaw.Cluster {
			removeCluster = false
		}
	}

	// Remove AuthInfo if not used by any other context
	if removeAuthInfo {
		delete(kubeConfig.AuthInfos, contextRaw.AuthInfo)
	}

	// Remove Cluster if not used by any other context
	if removeCluster {
		delete(kubeConfig.Clusters, contextRaw.Cluster)
	}

	if kubeConfig.CurrentContext == kubeContext {
		kubeConfig.CurrentContext = ""

		if len(kubeConfig.Contexts) > 0 {
			for context, contextObj := range kubeConfig.Contexts {
				if contextObj != nil {
					kubeConfig.CurrentContext = context
					break
				}
			}
		}
	}

	return nil
}
