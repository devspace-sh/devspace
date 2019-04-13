package cloud

import (
	"fmt"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/pkg/errors"
)

// Space holds the information about a space in the cloud
type Space struct {
	SpaceID      int      `yaml:"spaceID"`
	Name         string   `yaml:"name"`
	Owner        *Owner   `yaml:"account"`
	ProviderName string   `yaml:"providerName"`
	Cluster      *Cluster `yaml:"cluster"`
	Created      string   `yaml:"created"`
	Domain       *string  `yaml:"domain"`
}

// ServiceAccount holds the information about a service account for a certain space
type ServiceAccount struct {
	SpaceID   int    `yaml:"spaceID"`
	Namespace string `yaml:"namespace"`
	CaCert    string `yaml:"caCert"`
	Server    string `yaml:"server"`
	Token     string `yaml:"token"`
}

// Project is the type that holds the project information
type Project struct {
	ProjectID int      `json:"id"`
	OwnerID   int      `json:"owner_id"`
	Cluster   *Cluster `json:"cluster"`
	Name      string   `json:"name"`
}

// Cluster is the type that holds the cluster information
type Cluster struct {
	ClusterID int     `json:"id"`
	Server    *string `json:"server"`
	Owner     *Owner  `json:"account"`
	Name      string  `json:"name"`
	CreatedAt *string `json:"created_at"`
}

// Owner holds the information about a certain
type Owner struct {
	OwnerID int    `json:"id"`
	Name    string `json:"name"`
}

// ClusterUser is the type that golds the cluster user information
type ClusterUser struct {
	ClusterUserID int  `json:"id"`
	AccountID     int  `json:"account_id"`
	ClusterID     int  `json:"cluster_id"`
	IsAdmin       bool `json:"is_admin"`
}

// Registry is the type that holds the docker image registry information
type Registry struct {
	RegistryID int    `json:"id"`
	URL        string `json:"url"`
	OwnerID    *int   `json:"owner_id"`
}

// GetRegistries returns all docker image registries
func (p *Provider) GetRegistries() ([]*Registry, error) {
	// Response struct
	response := struct {
		ImageRegistry []*Registry `json:"image_registry"`
	}{}

	// Do the request
	err := p.GrapqhlRequest(`
		query {
			image_registry {
				id
				url
				owner_id
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
func (p *Provider) GetClusterByName(clusterName string) (*Cluster, error) {
	clusterNameSplitted := strings.Split(clusterName, ":")
	if len(clusterNameSplitted) > 2 {
		return nil, fmt.Errorf("Error parsing cluster name %s: Expected : only once", clusterName)
	}

	clusterName = clusterNameSplitted[0]
	accountName, err := token.GetAccountName(p.Token)
	if err != nil {
		return nil, errors.Wrap(err, "get account name")
	}
	if len(clusterNameSplitted) == 2 {
		accountName = clusterNameSplitted[0]
		clusterName = clusterNameSplitted[1]
	}

	// Response struct
	response := struct {
		Clusters []*Cluster `json:"cluster"`
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
func (p *Provider) GetClusters() ([]*Cluster, error) {
	// Response struct
	response := struct {
		Clusters []*Cluster `json:"cluster"`
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
func (p *Provider) GetProjects() ([]*Project, error) {
	// Response struct
	response := struct {
		Projects []*Project `json:"project"`
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
func (p *Provider) GetClusterUser(clusterID int) (*ClusterUser, error) {
	// Response struct
	response := struct {
		ClusterUser []*ClusterUser `json:"cluster_user"`
	}{}

	// Get account id
	accountID, err := token.GetAccountID(p.Token)
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
func (p *Provider) GetServiceAccount(space *Space) (*ServiceAccount, error) {
	key, err := p.GetClusterKey(space.Cluster)
	if err != nil {
		return nil, errors.Wrap(err, "get cluster key")
	}

	// Response struct
	response := struct {
		ServiceAccount *ServiceAccount `json:"manager_serviceAccount"`
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
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Owner *Owner `json:"account"`

	KubeContext *struct {
		Cluster *Cluster `json:"cluster"`
		Domains []*struct {
			URL string `json:"url"`
		} `json:"kube_context_domains"`
	} `json:"kube_context"`

	CreatedAt string `json:"created_at"`
}

// GetSpaces returns all spaces by the user
func (p *Provider) GetSpaces() ([]*Space, error) {
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
				cluster {
					id
					name

					account {
						id
						name
					}
				}
				
				kube_context_domains(limit:1) {
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

	retSpaces := []*Space{}
	for _, spaceConfig := range response.Spaces {
		if spaceConfig.KubeContext == nil {
			return nil, fmt.Errorf("KubeContext is nil for space %s", spaceConfig.Name)
		}

		newSpace := &Space{
			SpaceID:      spaceConfig.ID,
			Owner:        spaceConfig.Owner,
			Name:         spaceConfig.Name,
			ProviderName: p.Name,
			Cluster:      spaceConfig.KubeContext.Cluster,
			Created:      spaceConfig.CreatedAt,
		}
		if spaceConfig.KubeContext.Domains != nil && len(spaceConfig.KubeContext.Domains) > 0 {
			newSpace.Domain = &spaceConfig.KubeContext.Domains[0].URL
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
func (p *Provider) GetSpace(spaceID int) (*Space, error) {
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
				cluster {
					id
					name
					account {
						id
						name
					}
				}

				kube_context_domains(limit:1) {
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

	retSpace := &Space{
		SpaceID:      spaceConfig.ID,
		Owner:        spaceConfig.Owner,
		Name:         spaceConfig.Name,
		ProviderName: p.Name,
		Cluster:      spaceConfig.KubeContext.Cluster,
		Created:      spaceConfig.CreatedAt,
	}
	if spaceConfig.KubeContext.Domains != nil && len(spaceConfig.KubeContext.Domains) > 0 {
		retSpace.Domain = &spaceConfig.KubeContext.Domains[0].URL
	}

	// Exchange space name
	err = p.exchangeSpaceName(retSpace)
	if err != nil {
		return nil, err
	}

	return retSpace, nil
}

// GetSpaceByName returns a space by name
func (p *Provider) GetSpaceByName(spaceName string) (*Space, error) {
	spaceNameSplitted := strings.Split(spaceName, ":")
	if len(spaceNameSplitted) > 2 {
		return nil, fmt.Errorf("Error parsing space name %s: Expected : only once", spaceName)
	}

	spaceName = spaceNameSplitted[0]
	accountName, err := token.GetAccountName(p.Token)
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
					cluster {
						id
						name
						account {
							id
							name
						}
					}

					kube_context_domains(limit:1) {
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

	retSpace := &Space{
		SpaceID:      spaceConfig.ID,
		Owner:        spaceConfig.Owner,
		Name:         spaceConfig.Name,
		ProviderName: p.Name,
		Cluster:      spaceConfig.KubeContext.Cluster,
		Created:      spaceConfig.CreatedAt,
	}
	if spaceConfig.KubeContext.Domains != nil && len(spaceConfig.KubeContext.Domains) > 0 {
		retSpace.Domain = &spaceConfig.KubeContext.Domains[0].URL
	}

	// Exchange space name
	err = p.exchangeSpaceName(retSpace)
	if err != nil {
		return nil, err
	}

	return retSpace, nil
}

func (p *Provider) exchangeSpaceName(space *Space) error {
	userAccountName, err := token.GetAccountName(p.Token)
	if err != nil {
		return errors.Wrap(err, "get account name")
	}

	if space.Owner.Name != userAccountName {
		space.Name = space.Owner.Name + ":" + space.Name
	}

	// Exchange also cluster name
	return p.exchangeClusterName(space.Cluster)
}

func (p *Provider) exchangeClusterName(cluster *Cluster) error {
	if cluster.Owner == nil {
		return nil
	}

	userAccountName, err := token.GetAccountName(p.Token)
	if err != nil {
		return errors.Wrap(err, "get account name")
	}

	if cluster.Owner.Name != userAccountName {
		cluster.Name = cluster.Owner.Name + ":" + cluster.Name
	}

	return nil
}
