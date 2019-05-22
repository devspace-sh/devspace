package kubeconfig

import (
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// ConfigExists checks if a kube config exists
func ConfigExists() bool {
	return clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename() != ""
}

// LoadConfig loads the kube config with the default loading rules
func LoadConfig() clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
}

// LoadConfigFromContext loads the kube client config from a certain context
func LoadConfigFromContext(context string) (clientcmd.ClientConfig, error) {
	kubeConfig, err := LoadRawConfig()
	if err != nil {
		return nil, err
	}

	return clientcmd.NewNonInteractiveClientConfig(*kubeConfig, context, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules()), nil
}

// LoadRawConfig loads the raw kube config with the default loading rules
func LoadRawConfig() (*api.Config, error) {
	config, err := LoadConfig().RawConfig()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig writes the kube config back to the specified filename
func SaveConfig(config *api.Config) error {
	return clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), *config, false)
}
