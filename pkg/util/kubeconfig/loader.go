package kubeconfig

import (
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// Loader loads the kubeconfig
type Loader interface {
	ConfigExists() bool
	NewConfig() clientcmd.ClientConfig
	LoadConfig() clientcmd.ClientConfig
	LoadConfigFromContext(context string) (clientcmd.ClientConfig, error)
	LoadRawConfig() (*api.Config, error)

	GetCurrentContext() (string, error)
	GetCurrentNamespace() (string, error)

	SaveConfig(config *api.Config) error

	LoadNewConfig(contextName, server, caCert, token, namespace string) (clientcmd.ClientConfig, error)

	IsCloudSpace(context string) (bool, error)
	GetSpaceID(context string) (int, string, error)

	DeleteKubeContext(kubeConfig *api.Config, kubeContext string) error
}

type loader struct {
}

// NewLoader creates a new instance of the interface Loader
func NewLoader() Loader {
	return &loader{}
}
