package cloud

import (
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"

	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"
)

// DeleteCluster deletes an cluster
func (p *Provider) DeleteCluster(cluster *Cluster, deleteServices, deleteKubeContexts bool) error {
	key, err := p.GetClusterKey(cluster)
	if err != nil {
		return errors.Wrap(err, "get cluster key")
	}

	err = p.GrapqhlRequest(`
		mutation($key:String!,$clusterID:Int!,$deleteServices:Boolean!,$deleteKubeContexts:Boolean!){
			manager_deleteCluster(
				key:$key,
				clusterID:$clusterID,
				deleteServices:$deleteServices,
				deleteKubeContexts:$deleteKubeContexts
			)
		}
	`, map[string]interface{}{
		"key":                key,
		"clusterID":          cluster.ClusterID,
		"deleteServices":     deleteServices,
		"deleteKubeContexts": deleteKubeContexts,
	}, &struct {
		DeleteCluster bool `json:"manager_deleteCluster"`
	}{})
	if err != nil {
		return err
	}

	return nil
}

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
func DeleteKubeContext(space *Space) error {
	config, err := kubeconfig.ReadKubeConfig(clientcmd.RecommendedHomeFile)
	if err != nil {
		return err
	}

	hasChanged := false
	kubeContext := GetKubeContextNameFromSpace(space.Name, space.ProviderName)

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
