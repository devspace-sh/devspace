package cloud

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
)

// DeleteKubeContext removes the specified devspace id from the kube context if it exists
func (p *provider) DeleteKubeContext(space *latest.Space) error {
	kubeContext := GetKubeContextNameFromSpace(space.Name, space.ProviderName)
	kubeConfig, err := kubeconfig.LoadRawConfig()
	if err != nil {
		return err
	}

	err = kubeconfig.DeleteKubeContext(kubeConfig, kubeContext)
	if err != nil {
		return err
	}

	return kubeconfig.SaveConfig(kubeConfig)
}
