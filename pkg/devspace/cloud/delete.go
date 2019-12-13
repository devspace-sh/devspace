package cloud

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/pkg/errors"
)

// DeleteKubeContext removes the specified space from the kube context and providers.yaml
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

	err = kubeconfig.SaveConfig(kubeConfig)
	if err != nil {
		return errors.Wrap(err, "save kube config")
	}

	providerConfig, err := p.loader.Load()
	if err != nil {
		return errors.Wrap(err, "load provider config")
	}

	for _, profile := range providerConfig.Providers {
		for id := range profile.Spaces {
			if id == space.SpaceID {
				delete(profile.Spaces, id)
			}
		}
	}

	err = p.loader.Save(providerConfig)
	if err != nil {
		return errors.Wrap(err, "save provider config")
	}

	return nil
}
