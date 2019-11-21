package client

import (
	"fmt"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/pkg/errors"
)

// GetRegistries returns all docker image registries
func (c *client) GetRegistries() ([]*latest.Registry, error) {
	// Response struct
	response := struct {
		ImageRegistry []*latest.Registry `json:"image_registry"`
	}{}

	// Do the request
	err := c.grapqhlRequest(`
		query {
			image_registry {
				id
				url
			}
		}
	`, nil, &response)
	if err != nil {
		return nil, err
	}

	// Check result
	if response.ImageRegistry == nil {
		return nil, errors.New("Wrong answer from graphql server: ImageRegistries is nil")
	}

	return response.ImageRegistry, nil
}

// GetClusterByName retrieves an user cluster by name (username:clustername)
func (c *client) GetClusterByName(clusterName string) (*latest.Cluster, error) {
	clusterNameSplitted := strings.Split(clusterName, ":")
	if len(clusterNameSplitted) > 2 {
		return nil, errors.Errorf("Error parsing cluster name %s: Expected : only once", clusterName)
	}

	baererToken, err := c.GetToken()
	if err != nil {
		return nil, errors.Wrap(err, "get token")
	}

	clusterName = clusterNameSplitted[0]

	accountName, err := token.GetAccountName(baererToken)
	if err != nil {
		return nil, errors.Wrap(err, "get account name")
	}
	if len(clusterNameSplitted) == 2 {
		accountName = clusterNameSplitted[0]
		clusterName = clusterNameSplitted[1]
	}

	// Response struct
	response := struct {
		Clusters []*latest.Cluster `json:"cluster"`
	}{}

	err = c.grapqhlRequest(`
	query ($accountName:String!, $clusterName:String!){
		cluster (where:{_and:[
			{name: {_eq:$clusterName}},
			{_or: [
				{owner_id: {_is_null:true}},
				{account: {name: {_eq:$accountName}}}
			]}
		]},limit:1){
			id
			account {
				id
				name
			}

			encrypt_token
			name
			server
		}
	}
  `, map[string]interface{}{
		"accountName": accountName,
		"clusterName": clusterName,
	}, &response)
	if err != nil {
		return nil, err
	}
	if len(response.Clusters) != 1 {
		return nil, errors.Errorf("Couldn't find cluster for cluster name %s", clusterName)
	}

	// Exchange cluster name
	err = c.exchangeClusterName(response.Clusters[0])
	if err != nil {
		return nil, err
	}

	return response.Clusters[0], nil
}

// GetClusters returns all clusters accessable by the user
func (c *client) GetClusters() ([]*latest.Cluster, error) {
	// Response struct
	response := struct {
		Clusters []*latest.Cluster `json:"cluster"`
	}{}

	// Do the request
	err := c.grapqhlRequest(`
	  query {
		cluster {
			id
			account {
				id
				name
			}

			name
			encrypt_token
			server
			created_at
		}
	  }
	`, nil, &response)
	if err != nil {
		return nil, err
	}

	// Check result
	if response.Clusters == nil {
		return nil, errors.New("Wrong answer from graphql server: Clusters is nil")
	}

	// Exchange cluster name
	for _, cluster := range response.Clusters {
		err := c.exchangeClusterName(cluster)
		if err != nil {
			return nil, err
		}
	}

	return response.Clusters, nil
}

// GetProjects returns all projects by the user
func (c *client) GetProjects() ([]*latest.Project, error) {
	// Response struct
	response := struct {
		Projects []*latest.Project `json:"project"`
	}{}

	// Do the request
	err := c.grapqhlRequest(`
	  query {
		project {
			id
			owner_id
			cluster_id
			name
		}
	  }
	`, nil, &response)
	if err != nil {
		return nil, err
	}

	// Check result
	if response.Projects == nil {
		return nil, errors.New("Wrong answer from graphql server: Projects is nil")
	}

	return response.Projects, nil
}

// GetClusterUser retrieves the cluster user
func (c *client) GetClusterUser(clusterID int) (*latest.ClusterUser, error) {
	// Response struct
	response := struct {
		ClusterUser []*latest.ClusterUser `json:"cluster_user"`
	}{}

	baererToken, err := c.GetToken()
	if err != nil {
		return nil, errors.Wrap(err, "get token")
	}

	// Get account id
	accountID, err := token.GetAccountID(baererToken)
	if err != nil {
		return nil, err
	}

	// Do the request
	err = c.grapqhlRequest(`
		query ($clusterID: Int!, $accountID: Int!) {
		cluster_user(where:{_and:[
			{cluster_id:{_eq:$clusterID}},
			{account_id:{_eq:$accountID}}
		]}) {
				id
				account_id
				cluster_id
				is_admin
			}
		}
	`, map[string]interface{}{
		"clusterID": clusterID,
		"accountID": accountID,
	}, &response)
	if err != nil {
		return nil, err
	}
	if len(response.ClusterUser) != 1 {
		return nil, errors.Errorf("Couldn't find cluster user for cluster %d", clusterID)
	}

	return response.ClusterUser[0], nil
}

// GetServiceAccount returns a service account for a certain space
func (c *client) GetServiceAccount(space *latest.Space, key string) (*latest.ServiceAccount, error) {
	// Response struct
	response := struct {
		ServiceAccount *latest.ServiceAccount `json:"manager_serviceAccount"`
	}{}

	// Do the request
	err := c.grapqhlRequest(`
	  query($spaceID:Int!, $key: String) {
		manager_serviceAccount(spaceID:$spaceID, key: $key) {
		  namespace
		  caCert
		  server
		  token
		}
	  }
	`, map[string]interface{}{
		"spaceID": space.SpaceID,
		"key":     key,
	}, &response)
	if err != nil {
		return nil, err
	}

	response.ServiceAccount.SpaceID = space.SpaceID
	return response.ServiceAccount, nil
}

type spaceGraphql struct {
	ID    int           `json:"id"`
	Name  string        `json:"name"`
	Owner *latest.Owner `json:"account"`

	KubeContext *struct {
		Namespace string          `json:"namespace"`
		Cluster   *latest.Cluster `json:"cluster"`
	} `json:"kube_context"`

	CreatedAt string `json:"created_at"`
}

// GetSpaces returns all spaces by the user
func (c *client) GetSpaces() ([]*latest.Space, error) {
	// Response struct
	response := struct {
		Spaces []*spaceGraphql `json:"space"`
	}{}

	// Do the request
	err := c.grapqhlRequest(`
	  query {
		space {
			id
			name
			account {
				id
				name
			}

			kube_context {
				namespace

				cluster {
					id
					name
					encrypt_token
					account {
						id
						name
					}
				}
			}
			
			created_at
		}
	  }
	`, nil, &response)
	if err != nil {
		return nil, err
	}

	// Check result
	if response.Spaces == nil {
		return nil, errors.New("Wrong answer from graphql server: Spaces is nil")
	}

	retSpaces := []*latest.Space{}
	for _, spaceConfig := range response.Spaces {
		if spaceConfig.KubeContext == nil {
			return nil, errors.Errorf("KubeContext is nil for space %s", spaceConfig.Name)
		}

		newSpace := &latest.Space{
			SpaceID:      spaceConfig.ID,
			Owner:        spaceConfig.Owner,
			Name:         spaceConfig.Name,
			Namespace:    spaceConfig.KubeContext.Namespace,
			ProviderName: c.provider,
			Cluster:      spaceConfig.KubeContext.Cluster,
			Created:      spaceConfig.CreatedAt,
		}

		// Exchange space name
		err = c.exchangeSpaceName(newSpace)
		if err != nil {
			return nil, err
		}

		retSpaces = append(retSpaces, newSpace)
	}

	return retSpaces, nil
}

// GetSpace returns a specific space by id
func (c *client) GetSpace(spaceID int) (*latest.Space, error) {
	// Response struct
	response := struct {
		Space *spaceGraphql `json:"space_by_pk"`
	}{}

	// Do the request
	err := c.grapqhlRequest(`
	  query($ID:Int!) {
		space_by_pk(id:$ID) {
			id
			name
			account {
				id
				name
			}
			
			kube_context {
				namespace
				cluster {
					id
					name
					encrypt_token
					account {
						id
						name
					}
				}
			}
			
			created_at
		}
	  }
	`, map[string]interface{}{
		"ID": spaceID,
	}, &response)
	if err != nil {
		return nil, err
	}

	// Check result
	if response.Space == nil {
		return nil, errors.Errorf("Space %d not found", spaceID)
	}

	spaceConfig := response.Space
	if spaceConfig.KubeContext == nil {
		return nil, errors.Errorf("KubeContext is nil for space %s", spaceConfig.Name)
	}

	retSpace := &latest.Space{
		SpaceID:      spaceConfig.ID,
		Owner:        spaceConfig.Owner,
		Name:         spaceConfig.Name,
		Namespace:    spaceConfig.KubeContext.Namespace,
		ProviderName: c.provider,
		Cluster:      spaceConfig.KubeContext.Cluster,
		Created:      spaceConfig.CreatedAt,
	}

	// Exchange space name
	err = c.exchangeSpaceName(retSpace)
	if err != nil {
		return nil, err
	}

	return retSpace, nil
}

// GetSpaceByName returns a space by name
func (c *client) GetSpaceByName(spaceName string) (*latest.Space, error) {
	spaceNameSplitted := strings.Split(spaceName, ":")
	if len(spaceNameSplitted) > 2 {
		return nil, errors.Errorf("Error parsing space name %s: Expected : only once", spaceName)
	}

	baererToken, err := c.GetToken()
	if err != nil {
		return nil, errors.Wrap(err, "get token")
	}

	spaceName = spaceNameSplitted[0]

	accountName, err := token.GetAccountName(baererToken)
	if err != nil {
		return nil, errors.Wrap(err, "get account name")
	}
	if len(spaceNameSplitted) == 2 {
		accountName = spaceNameSplitted[0]
		spaceName = spaceNameSplitted[1]
	}

	// Response struct
	response := struct {
		Space []*spaceGraphql `json:"space"`
	}{}

	// Do the request
	err = c.grapqhlRequest(`
		query($accountName:String!, $spaceName:String!) {
			space(where:{
			_and: [
				{account: {name:{_eq:$accountName}}},
				{name: {_eq:$spaceName}}
			]},limit:1){
				id
				name
				account {
					id
					name
				}
				
				kube_context {
					namespace
					cluster {
						id
						name
						encrypt_token
						account {
							id
							name
						}
					}
				}
				
				created_at
			}
		}
	`, map[string]interface{}{
		"spaceName":   spaceName,
		"accountName": accountName,
	}, &response)
	if err != nil {
		return nil, err
	}

	// Check result
	if response.Space == nil || len(response.Space) == 0 {
		return nil, fmt.Errorf(message.SpaceNotFound, spaceName)
	}

	spaceConfig := response.Space[0]
	if spaceConfig.KubeContext == nil {
		return nil, errors.Errorf("KubeContext is nil for space %s", spaceConfig.Name)
	}

	retSpace := &latest.Space{
		SpaceID:      spaceConfig.ID,
		Owner:        spaceConfig.Owner,
		Name:         spaceConfig.Name,
		Namespace:    spaceConfig.KubeContext.Namespace,
		ProviderName: c.provider,
		Cluster:      spaceConfig.KubeContext.Cluster,
		Created:      spaceConfig.CreatedAt,
	}

	// Exchange space name
	err = c.exchangeSpaceName(retSpace)
	if err != nil {
		return nil, err
	}

	return retSpace, nil
}

func (c *client) exchangeSpaceName(space *latest.Space) error {
	baererToken, err := c.GetToken()
	if err != nil {
		return errors.Wrap(err, "get token")
	}

	userAccountName, err := token.GetAccountName(baererToken)
	if err != nil {
		return errors.Wrap(err, "get account name")
	}

	if space.Owner.Name != userAccountName {
		space.Name = space.Owner.Name + ":" + space.Name
	}

	// Exchange also cluster name
	return c.exchangeClusterName(space.Cluster)
}

func (c *client) exchangeClusterName(cluster *latest.Cluster) error {
	if cluster.Owner == nil {
		return nil
	}

	baererToken, err := c.GetToken()
	if err != nil {
		return errors.Wrap(err, "get token")
	}

	userAccountName, err := token.GetAccountName(baererToken)
	if err != nil {
		return errors.Wrap(err, "get account name")
	}

	if cluster.Owner.Name != userAccountName {
		cluster.Name = cluster.Owner.Name + ":" + cluster.Name
	}

	return nil
}

// VerifyKey verifies the given key for the given cluster
func (c *client) VerifyKey(clusterID int, key string) (bool, error) {
	// Response struct
	response := struct {
		VerifyKey bool `json:"manager_verifyUserClusterKey"`
	}{}

	// Do the request
	err := c.grapqhlRequest(`
		mutation ($clusterID:Int!, $key:String!) {
			manager_verifyUserClusterKey(
				clusterID: $clusterID,
				key: $key
			)
		}
	`, map[string]interface{}{
		"clusterID": clusterID,
		"key":       key,
	}, &response)
	if err != nil {
		return false, err
	}

	return response.VerifyKey, nil
}

//Setting is a setting object in a server response
type Setting struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// Settings retrieves cloud settings
func (c *client) Settings(encryptToken string) ([]Setting, error) {
	// Response struct
	response := struct {
		Settings []Setting `json:"manager_settings"`
	}{}

	// Do the request
	err := c.grapqhlRequest(`
		query ($settings: [String!]!) {
			manager_settings(settings:$settings) {
				id
				value
			}
		}
	`, map[string]interface{}{
		"settings": []string{encryptToken},
	}, &response)
	if err != nil {
		return nil, err
	}

	return response.Settings, nil
}
