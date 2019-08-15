package kubeconfig

import (
	"encoding/base64"

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

// LoadNewConfig creates a new config from scratch with the given parameters and loads it
func LoadNewConfig(contextName, server, caCert, token, namespace string) (clientcmd.ClientConfig, error) {
	config := api.NewConfig()
	decodedCaCert, err := base64.StdEncoding.DecodeString(caCert)
	if err != nil {
		return nil, err
	}

	cluster := api.NewCluster()
	cluster.Server = server
	cluster.CertificateAuthorityData = decodedCaCert

	authInfo := api.NewAuthInfo()
	authInfo.Token = token

	config.Clusters[contextName] = cluster
	config.AuthInfos[contextName] = authInfo

	// Update kube context
	context := api.NewContext()
	context.Cluster = contextName
	context.AuthInfo = contextName

	if namespace != "" {
		context.Namespace = namespace
	}

	config.Contexts[contextName] = context
	config.CurrentContext = contextName

	return clientcmd.NewNonInteractiveClientConfig(*config, contextName, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules()), nil
}
