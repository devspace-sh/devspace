package cloud

import (
	"errors"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"k8s.io/client-go/tools/clientcmd"
)

// DeleteSpace deletes a space with the given id
func (p *Provider) DeleteSpace(spaceID int) error {
	// Response struct
	response := struct {
		ManagerDeleteSpace bool `json:"manager_deleteSpace"`
	}{}

	// Do the request
	err := p.GrapqhlRequest(`
		mutation($spaceID: Int!) {
			manager_deleteSpace(spaceID: $spaceID)
		}
	`, map[string]interface{}{
		"spaceID": spaceID,
	}, &response)
	if err != nil {
		return err
	}

	// Check result
	if response.ManagerDeleteSpace == false {
		return errors.New("Mutation returned wrong result")
	}

	return nil
}

// DeleteKubeContext removes the specified devspace id from the kube context if it exists
func DeleteKubeContext(space *generated.SpaceConfig) error {
	config, err := kubeconfig.ReadKubeConfig(clientcmd.RecommendedHomeFile)
	if err != nil {
		return err
	}

	hasChanged := false
	kubeContext := GetKubeContextNameFromSpace(space)

	if _, ok := config.Clusters[kubeContext]; ok {
		delete(config.Clusters, kubeContext)
		hasChanged = true
	}

	if _, ok := config.AuthInfos[kubeContext]; ok {
		delete(config.AuthInfos, kubeContext)
		hasChanged = true
	}

	if _, ok := config.Contexts[kubeContext]; ok {
		delete(config.Contexts, kubeContext)
		hasChanged = true
	}

	if config.CurrentContext == kubeContext {
		config.CurrentContext = ""

		if len(config.Contexts) > 0 {
			for context, contextObj := range config.Contexts {
				if contextObj != nil {
					config.CurrentContext = context
					break
				}
			}
		}

		hasChanged = true
	}

	if hasChanged {
		return kubeconfig.WriteKubeConfig(config, clientcmd.RecommendedHomeFile)
	}

	return nil
}
