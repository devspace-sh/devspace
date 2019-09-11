package cloud

import (
	"fmt"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

// GetRegistries returns all docker image registries
func (p *Provider) GetRegistries() ([]*latest.Registry, error) {
	// Response struct
	response := struct {
		ImageRegistry []*latest.Registry `json:"image_registry"`
	}{}

	// Do the request
	err := p.GrapqhlRequest(`
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
func (p *Provider) GetClusterByName(clusterName string) (*latest.Cluster, error) {
	clusterNameSplitted := strings.Split(clusterName, ":")
	if len(clusterNameSplitted) > 2 {
		return nil, fmt.Errorf("Error parsing cluster name %s: Expected : only once", clusterName)
	}

	clusterName = clusterNameSplitted[0]

	bearerToken, err := p.GetToken()
	if err != nil {
		return nil, errors.Wrap(err, "get token")
	}

	accountName, err := token.GetAccountName(bearerToken)
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

	err = p.GrapqhlRequest(`
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
		return nil, fmt.Errorf("Couldn't find cluster for cluster name %s", clusterName)
	}

	// Exchange cluster name
	err = p.exchangeClusterName(response.Clusters[0])
	if err != nil {
		return nil, err
	}

	return response.Clusters[0], nil
}

// GetClusters returns all clusters accessable by the user
func (p *Provider) GetClusters() ([]*latest.Cluster, error) {
	// Response struct
	response := struct {
		Clusters []*latest.Cluster `json:"cluster"`
	}{}

	// Do the request
	err := p.GrapqhlRequest(`
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
		err := p.exchangeClusterName(cluster)
		if err != nil {
			return nil, err
		}
	}

	return response.Clusters, nil
}

// GetProjects returns all projects by the user
func (p *Provider) GetProjects() ([]*latest.Project, error) {
	// Response struct
	response := struct {
		Projects []*latest.Project `json:"project"`
	}{}

	// Do the request
	err := p.GrapqhlRequest(`
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
func (p *Provider) GetClusterUser(clusterID int) (*latest.ClusterUser, error) {
	// Response struct
	response := struct {
		ClusterUser []*latest.ClusterUser `json:"cluster_user"`
	}{}

	bearerToken, err := p.GetToken()
	if err != nil {
		return nil, errors.Wrap(err, "get token")
	}

	// Get account id
	accountID, err := token.GetAccountID(bearerToken)
	if err != nil {
		return nil, err
	}

	// Do the request
	err = p.GrapqhlRequest(`
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
		return nil, fmt.Errorf("Couldn't find cluster user for cluster %d", clusterID)
	}

	return response.ClusterUser[0], nil
}

// GetServiceAccount returns a service account for a certain space
func (p *Provider) GetServiceAccount(space *latest.Space, log log.Logger) (*latest.ServiceAccount, error) {
	key, err := p.GetClusterKey(space.Cluster, log)
	if err != nil {
		return nil, errors.Wrap(err, "get cluster key")
	}

	// Response struct
	response := struct {
		ServiceAccount *latest.ServiceAccount `json:"manager_serviceAccount"`
	}{}

	// Do the request
	err = p.GrapqhlRequest(`
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
		Namespace string                `json:"namespace"`
		Cluster   *latest.Cluster       `json:"cluster"`
		Domains   []*latest.SpaceDomain `json:"kube_context_domains"`
	} `json:"kube_context"`

	CreatedAt string `json:"created_at"`
}

// GetSpaces returns all spaces by the user
func (p *Provider) GetSpaces() ([]*latest.Space, error) {
	// Response struct
	response := struct {
		Spaces []*spaceGraphql `json:"space"`
	}{}

	// Do the request
	err := p.GrapqhlRequest(`
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
				
				kube_context_domains {
					id
					url
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
			return nil, fmt.Errorf("KubeContext is nil for space %s", spaceConfig.Name)
		}

		newSpace := &latest.Space{
			SpaceID:      spaceConfig.ID,
			Owner:        spaceConfig.Owner,
			Name:         spaceConfig.Name,
			Namespace:    spaceConfig.KubeContext.Namespace,
			ProviderName: p.Name,
			Cluster:      spaceConfig.KubeContext.Cluster,
			Domains:      spaceConfig.KubeContext.Domains,
			Created:      spaceConfig.CreatedAt,
		}

		// Exchange space name
		err = p.exchangeSpaceName(newSpace)
		if err != nil {
			return nil, err
		}

		retSpaces = append(retSpaces, newSpace)
	}

	return retSpaces, nil
}

// GetSpace returns a specific space by id
func (p *Provider) GetSpace(spaceID int) (*latest.Space, error) {
	// Response struct
	response := struct {
		Space *spaceGraphql `json:"space_by_pk"`
	}{}

	// Do the request
	err := p.GrapqhlRequest(`
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

				kube_context_domains {
					id
					url
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
		return nil, fmt.Errorf("Space %d not found", spaceID)
	}

	spaceConfig := response.Space
	if spaceConfig.KubeContext == nil {
		return nil, fmt.Errorf("KubeContext is nil for space %s", spaceConfig.Name)
	}

	retSpace := &latest.Space{
		SpaceID:      spaceConfig.ID,
		Owner:        spaceConfig.Owner,
		Name:         spaceConfig.Name,
		Namespace:    spaceConfig.KubeContext.Namespace,
		ProviderName: p.Name,
		Cluster:      spaceConfig.KubeContext.Cluster,
		Domains:      spaceConfig.KubeContext.Domains,
		Created:      spaceConfig.CreatedAt,
	}

	// Exchange space name
	err = p.exchangeSpaceName(retSpace)
	if err != nil {
		return nil, err
	}

	return retSpace, nil
}

// GetSpaceByName returns a space by name
func (p *Provider) GetSpaceByName(spaceName string) (*latest.Space, error) {
	spaceNameSplitted := strings.Split(spaceName, ":")
	if len(spaceNameSplitted) > 2 {
		return nil, fmt.Errorf("Error parsing space name %s: Expected : only once", spaceName)
	}

	spaceName = spaceNameSplitted[0]

	bearerToken, err := p.GetToken()
	if err != nil {
		return nil, errors.Wrap(err, "get token")
	}

	accountName, err := token.GetAccountName(bearerToken)
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
	err = p.GrapqhlRequest(`
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

					kube_context_domains {
						id
						url
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
	if response.Space == nil {
		return nil, fmt.Errorf("Space %s not found", spaceName)
	}
	if len(response.Space) == 0 {
		return nil, fmt.Errorf("Space %s not found", spaceName)
	}

	spaceConfig := response.Space[0]
	if spaceConfig.KubeContext == nil {
		return nil, fmt.Errorf("KubeContext is nil for space %s", spaceConfig.Name)
	}

	retSpace := &latest.Space{
		SpaceID:      spaceConfig.ID,
		Owner:        spaceConfig.Owner,
		Name:         spaceConfig.Name,
		Namespace:    spaceConfig.KubeContext.Namespace,
		ProviderName: p.Name,
		Cluster:      spaceConfig.KubeContext.Cluster,
		Domains:      spaceConfig.KubeContext.Domains,
		Created:      spaceConfig.CreatedAt,
	}

	// Exchange space name
	err = p.exchangeSpaceName(retSpace)
	if err != nil {
		return nil, err
	}

	return retSpace, nil
}

func (p *Provider) exchangeSpaceName(space *latest.Space) error {
	bearerToken, err := p.GetToken()
	if err != nil {
		return errors.Wrap(err, "get token")
	}

	userAccountName, err := token.GetAccountName(bearerToken)
	if err != nil {
		return errors.Wrap(err, "get account name")
	}

	if space.Owner.Name != userAccountName {
		space.Name = space.Owner.Name + ":" + space.Name
	}

	// Exchange also cluster name
	return p.exchangeClusterName(space.Cluster)
}

func (p *Provider) exchangeClusterName(cluster *latest.Cluster) error {
	if cluster.Owner == nil {
		return nil
	}

	bearerToken, err := p.GetToken()
	if err != nil {
		return errors.Wrap(err, "get token")
	}

	userAccountName, err := token.GetAccountName(bearerToken)
	if err != nil {
		return errors.Wrap(err, "get account name")
	}

	if cluster.Owner.Name != userAccountName {
		cluster.Name = cluster.Owner.Name + ":" + cluster.Name
	}

	return nil
}
