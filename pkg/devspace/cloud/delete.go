package cloud

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/pkg/errors"
)

// DeleteKubeContext removes the specified space from the kube context and providers.yaml
func (p *provider) DeleteKubeContext(space *latest.Space) error {
	kubeContext := GetKubeContextNameFromSpace(space.Name, space.ProviderName)
	kubeConfig, err := p.kubeLoader.LoadRawConfig()
	if err != nil {
		return err
	}

	err = p.kubeLoader.DeleteKubeContext(kubeConfig, kubeContext)
	if err != nil {
		return err
	}

	err = p.kubeLoader.SaveConfig(kubeConfig)
	if err != nil {
		return errors.Wrap(err, "save kube config")
	}

	for id := range p.Spaces {
		if id == space.SpaceID {
			delete(p.Spaces, id)
		}
	}

	err = p.Save()
	if err != nil {
		return errors.Wrap(err, "save provider config")
	}

	return nil
}
