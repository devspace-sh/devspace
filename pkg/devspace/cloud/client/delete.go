package client

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
)

// DeleteCluster deletes an cluster
func (c *client) DeleteCluster(cluster *latest.Cluster, key string, deleteServices, deleteKubeContexts bool) error {
	err := c.grapqhlRequest(`
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
func (c *client) DeleteSpace(space *latest.Space, key string) (bool, error) {
	// Response struct
	response := struct {
		ManagerDeleteSpace bool `json:"manager_deleteSpace"`
	}{}

	// Do the request
	err := c.grapqhlRequest(`
		mutation($spaceID: Int!, $key: String!) {
			manager_deleteSpace(spaceID: $spaceID, key: $key)
		}
	`, map[string]interface{}{
		"spaceID": space.SpaceID,
		"key":     key,
	}, &response)
	if err != nil {
		return false, err
	}

	return response.ManagerDeleteSpace, nil
}
