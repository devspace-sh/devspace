package cloud

import (
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/util/kubeconfig"
	"k8s.io/client-go/tools/clientcmd"
)

// DeleteSpace deletes a space with the given id
func (p *Provider) DeleteSpace(spaceID int) error {
	panic("unimplemented")
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

/*
type managerDeleteDevSpaceMutation struct {
	ManagerDeleteDevSpace bool `json:"manager_deleteDevSpace"`
}

// DeleteDevSpace deletes the devspace from the cloud provider
func (p *Provider) DeleteDevSpace(devSpaceID int) error {
	// Delete kube contexts first
	targetConfigs, err := p.GetDevSpaceTargetConfigs(devSpaceID)
	if err != nil {
		return err
	}

	for _, targetConfig := range targetConfigs {
		err = DeleteKubeContext(targetConfig.Namespace)
		if err != nil {
			return fmt.Errorf("Error deleting kube context: %v", err)
		}
	}

	graphQlClient := graphql.NewClient(p.Host + GraphqlEndpoint)
	req := graphql.NewRequest(`
		mutation($devSpaceID: Int!) {
			manager_deleteDevSpace(devSpaceID: $devSpaceID)
		}
	`)

	req.Var("devSpaceID", devSpaceID)
	req.Header.Set("Authorization", p.Token)

	ctx := context.Background()
	response := managerDeleteDevSpaceMutation{}

	// Run the graphql request
	err = graphQlClient.Run(ctx, req, &response)
	if err != nil {
		return err
	}

	return nil
}

// DeleteKubeContext removes the specified devspace id from the kube context if it exists
func DeleteKubeContext(namespace string) error {
	config, err := kubeconfig.ReadKubeConfig(clientcmd.RecommendedHomeFile)
	if err != nil {
		return err
	}

	hasChanged := false
	kubeContext := DevSpaceKubeContextName + "-" + namespace

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
*/
