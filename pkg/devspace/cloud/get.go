package cloud

import (
	"errors"
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
)

// Project is the type that holds the project information
type Project struct {
	ProjectID int    `json:"id"`
	OwnerID   int    `json:"owner_id"`
	ClusterID int    `json:"cluster_id"`
	Name      string `json:"name"`
}

// Cluster is the type that holds the cluster information
type Cluster struct {
	ClusterID int     `json:"id"`
	OwnerID   *int    `json:"owner_id"`
	Name      *string `json:"name"`
	Server    string  `json:"server"`
	CaCert    string  `json:"ca_cert"`
}

// Registry is the type that holds the docker image registry information
type Registry struct {
	RegistryID int    `json:"id"`
	URL        string `json:"url"`
	OwnerID    *int   `json:"owner_id"`
}

// ClaimSet is the auth token claim set type
type ClaimSet struct {
	Subject string `json:"sub"`
}

// Token describes a JSON Web Token.
type Token struct {
	Raw       string
	Claims    *ClaimSet
	Signature []byte
}

// GetAccountName retrieves the account name for the current user
func (p *Provider) GetAccountName() (string, error) {
	token, err := ParseTokenClaims(p.Token)
	if err != nil {
		return "", err
	}

	return token.Claims.Subject, nil
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
		  owner_id
		  name
		  server
		  ca_cert
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

// GetSpaces returns all spaces by the user
func (p *Provider) GetSpaces() ([]*generated.SpaceConfig, error) {
	// Response struct
	response := struct {
		Spaces []*struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			KubeContext *struct {
				Namespace           string `json:"namespace"`
				ServiceAccountToken string `json:"service_account_token"`

				Cluster *struct {
					CaCert string `json:"ca_cert"`
					Server string `json:"server"`
				} `json:"clusterByclusterId"`

				Domains []*struct {
					URL string `json:"url"`
				} `json:"kubeContextDomainsBykubeContextId"`
			} `json:"kubeContextBykubeContextId"`
			CreatedAt string `json:"created_at"`
		} `json:"space"`
	}{}

	// Do the request
	err := p.GrapqhlRequest(`
	  query {
		space {
		  id
		  name
		  
		  kubeContextBykubeContextId {
			namespace
			service_account_token
			
			clusterByclusterId {
			  ca_cert
			  server
			}
			
			kubeContextDomainsBykubeContextId(limit:1) {
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

	retSpaces := []*generated.SpaceConfig{}
	for _, spaceConfig := range response.Spaces {
		if spaceConfig.KubeContext == nil {
			return nil, fmt.Errorf("KubeContext is nil for space %s", spaceConfig.Name)
		}
		if spaceConfig.KubeContext.Cluster == nil {
			return nil, fmt.Errorf("Cluster is nil for space %s", spaceConfig.Name)
		}

		newSpace := &generated.SpaceConfig{
			SpaceID:             spaceConfig.ID,
			Name:                spaceConfig.Name,
			Namespace:           spaceConfig.KubeContext.Namespace,
			ServiceAccountToken: spaceConfig.KubeContext.ServiceAccountToken,
			Server:              spaceConfig.KubeContext.Cluster.Server,
			CaCert:              spaceConfig.KubeContext.Cluster.CaCert,
			ProviderName:        p.Name,
			Created:             spaceConfig.CreatedAt,
		}
		if spaceConfig.KubeContext.Domains != nil && len(spaceConfig.KubeContext.Domains) > 0 {
			newSpace.Domain = &spaceConfig.KubeContext.Domains[0].URL
		}

		retSpaces = append(retSpaces, newSpace)
	}

	return retSpaces, nil
}

// GetSpace returns a specific space by id
func (p *Provider) GetSpace(spaceID int) (*generated.SpaceConfig, error) {
	// Response struct
	response := struct {
		Space *struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			KubeContext *struct {
				Namespace           string `json:"namespace"`
				ServiceAccountToken string `json:"service_account_token"`

				Cluster *struct {
					CaCert string `json:"ca_cert"`
					Server string `json:"server"`
				} `json:"clusterByclusterId"`

				Domains []*struct {
					URL string `json:"url"`
				} `json:"kubeContextDomainsBykubeContextId"`
			} `json:"kubeContextBykubeContextId"`
			CreatedAt string `json:"created_at"`
		} `json:"space_by_pk"`
	}{}

	// Do the request
	err := p.GrapqhlRequest(`
	  query($ID:Int!) {
		space_by_pk(id:$ID) {
		  id
		  name
		  
		  kubeContextBykubeContextId {
			namespace
			service_account_token
			
			clusterByclusterId {
			  ca_cert
			  server
			}
			
			kubeContextDomainsBykubeContextId(limit:1) {
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
	if spaceConfig.KubeContext.Cluster == nil {
		return nil, fmt.Errorf("Cluster is nil for space %s", spaceConfig.Name)
	}

	retSpace := &generated.SpaceConfig{
		SpaceID:             spaceConfig.ID,
		Name:                spaceConfig.Name,
		Namespace:           spaceConfig.KubeContext.Namespace,
		ServiceAccountToken: spaceConfig.KubeContext.ServiceAccountToken,
		Server:              spaceConfig.KubeContext.Cluster.Server,
		CaCert:              spaceConfig.KubeContext.Cluster.CaCert,
		ProviderName:        p.Name,
		Created:             spaceConfig.CreatedAt,
	}
	if spaceConfig.KubeContext.Domains != nil && len(spaceConfig.KubeContext.Domains) > 0 {
		retSpace.Domain = &spaceConfig.KubeContext.Domains[0].URL
	}

	return retSpace, nil
}

// GetSpaceByName returns a space by name
func (p *Provider) GetSpaceByName(spaceName string) (*generated.SpaceConfig, error) {
	// Response struct
	response := struct {
		Space []*struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			KubeContext *struct {
				Namespace           string `json:"namespace"`
				ServiceAccountToken string `json:"service_account_token"`

				Cluster *struct {
					CaCert string `json:"ca_cert"`
					Server string `json:"server"`
				} `json:"clusterByclusterId"`

				Domains []*struct {
					URL string `json:"url"`
				} `json:"kubeContextDomainsBykubeContextId"`
			} `json:"kubeContextBykubeContextId"`
			CreatedAt string `json:"created_at"`
		} `json:"space"`
	}{}

	// Do the request
	err := p.GrapqhlRequest(`
	  query($name:String!) {
		space(where:{name:{_eq:$name}},limit:1){
		  id
		  name
		  
		  kubeContextBykubeContextId {
			namespace
			service_account_token
			
			clusterByclusterId {
			  ca_cert
			  server
			}
			
			kubeContextDomainsBykubeContextId(limit:1) {
			  url
			}
		  }
		  
		  created_at
		}
	  }
	`, map[string]interface{}{
		"name": spaceName,
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
	if spaceConfig.KubeContext.Cluster == nil {
		return nil, fmt.Errorf("Cluster is nil for space %s", spaceConfig.Name)
	}

	retSpace := &generated.SpaceConfig{
		SpaceID:             spaceConfig.ID,
		Name:                spaceConfig.Name,
		Namespace:           spaceConfig.KubeContext.Namespace,
		ServiceAccountToken: spaceConfig.KubeContext.ServiceAccountToken,
		Server:              spaceConfig.KubeContext.Cluster.Server,
		CaCert:              spaceConfig.KubeContext.Cluster.CaCert,
		ProviderName:        p.Name,
		Created:             spaceConfig.CreatedAt,
	}
	if spaceConfig.KubeContext.Domains != nil && len(spaceConfig.KubeContext.Domains) > 0 {
		retSpace.Domain = &spaceConfig.KubeContext.Domains[0].URL
	}

	return retSpace, nil
}
