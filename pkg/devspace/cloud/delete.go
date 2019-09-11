package cloud

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/pkg/errors"
)

// DeleteCluster deletes an cluster
func (p *Provider) DeleteCluster(cluster *latest.Cluster, deleteServices, deleteKubeContexts bool, log log.Logger) error {
	key, err := p.GetClusterKey(cluster, log)
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
func (p *Provider) DeleteSpace(space *latest.Space, log log.Logger) error {
	key, err := p.GetClusterKey(space.Cluster, log)
	if err != nil {
		return errors.Wrap(err, "get cluster key")
	}

	// Response struct
	response := struct {
		ManagerDeleteSpace bool `json:"manager_deleteSpace"`
	}{}

	// Do the request
	err = p.GrapqhlRequest(`
		mutation($spaceID: Int!, $key: String!) {
			manager_deleteSpace(spaceID: $spaceID, key: $key)
		}
	`, map[string]interface{}{
		"spaceID": space.SpaceID,
		"key":     key,
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
func DeleteKubeContext(space *latest.Space) error {
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
