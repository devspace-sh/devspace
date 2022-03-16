package kubeconfig

import (
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// Loader loads the kubeconfig
type Loader interface {
	NewConfig() clientcmd.ClientConfig
	LoadConfig() clientcmd.ClientConfig
	LoadRawConfig() (*api.Config, error)

	GetCurrentContext() (string, error)
	SaveConfig(config *api.Config) error
	DeleteKubeContext(kubeConfig *api.Config, kubeContext string) error
}

type loader struct {
}

// NewLoader creates a new instance of the interface Loader
func NewLoader() Loader {
	return &loader{}
}
