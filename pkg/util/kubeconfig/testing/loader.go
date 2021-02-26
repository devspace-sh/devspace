package testing

import (
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type Loader struct {
	Config    clientcmd.ClientConfig
	RawConfig *api.Config
}

// NewConfig is a fake implementation of the function
func (l *Loader) NewConfig() clientcmd.ClientConfig {
	return l.Config
}

// LoadConfig is a fake implementation of the function
func (l *Loader) LoadConfig() clientcmd.ClientConfig {
	return l.Config
}

// LoadRawConfig is a fake implementation of the function
func (l *Loader) LoadRawConfig() (*api.Config, error) {
	return l.RawConfig, nil
}

// GetCurrentContext is a fake implementation of the function
func (l *Loader) GetCurrentContext() (string, error) {
	return l.RawConfig.CurrentContext, nil
}

// SaveConfig is a fake implementation of the function
func (l *Loader) SaveConfig(config *api.Config) error {
	l.RawConfig = config
	return nil
}

// IsCloudSpace is a fake implementation of the function
func (l *Loader) IsCloudSpace(context string) (bool, error) {
	return context == "devspace", nil
}

// GetSpaceID is a fake implementation of the function
func (l *Loader) GetSpaceID(context string) (int, string, error) {
	return 1, "testProvider", nil
}

// DeleteKubeContext is a fake implementation of the function
func (l *Loader) DeleteKubeContext(kubeConfig *api.Config, kubeContext string) error {
	return kubeconfig.NewLoader().DeleteKubeContext(kubeConfig, kubeContext)
}
