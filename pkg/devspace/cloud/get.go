package cloud

import "github.com/covexo/devspace/pkg/devspace/config/generated"

// Project is the type that holds the project information
type Project struct {
	ProjectID int
	ClusterID int
	Name      string
}

// Cluster is the type that holds the cluster information
type Cluster struct {
	ClusterID int
	OwnerID   *int
	Name      *string
	Server    string
	CaCert    string
}

// Registry is the type that holds the docker image registry information
type Registry struct {
	RegistryID int
	URL        string
	OwnerID    *int
}

// GetAccountName retrieves the account name for the current user
func (p *Provider) GetAccountName() (string, error) {
	panic("unimplemented")
}

// GetRegistries returns all docker image registries
func (p *Provider) GetRegistries() ([]*Registry, error) {
	panic("unimplemented")
}

// GetRegistry returns a docker image registry
func (p *Provider) GetRegistry(url string) (*Registry, error) {
	panic("unimplemented")
}

// GetClusters returns all clusters accessable by the user
func (p *Provider) GetClusters() ([]*Cluster, error) {
	panic("unimplemented")
}

// GetProjects returns all projects by the user
func (p *Provider) GetProjects() ([]*Project, error) {
	panic("unimplemented")
}

// GetSpaces returns all spaces by the user
func (p *Provider) GetSpaces() ([]*generated.SpaceConfig, error) {
	panic("unimplemented")
}

// GetSpace returns a specific space by id
func (p *Provider) GetSpace(spaceID int) (*generated.SpaceConfig, error) {
	panic("unimplemented")
}

// GetSpaceByName returns a space by name
func (p *Provider) GetSpaceByName(spaceName string) (*generated.SpaceConfig, error) {
	panic("unimplemented")
}

/*
// DevSpaceConfig holds the information of a devspace
type DevSpaceConfig struct {
	DevSpaceID int
	Name       string
	Created    string
}

// GetDevSpaces returns all devspaces owned by the user
func (p *Provider) GetSpaces() ([]*DevSpaceConfig, error) {
	graphQlClient := graphql.NewClient(p.Host + GraphqlEndpoint)
	req := graphql.NewRequest(`
		query {
			DevSpaces {
				DevSpaceID
				Name
				Created
			}
		}
	`)

	req.Header.Set("Authorization", p.Token)

	ctx := context.Background()
	response := devSpaceConfigQuery{}

	// Run the graphql request
	err := graphQlClient.Run(ctx, req, &response)
	if err != nil {
		return nil, err
	}

	return response.DevSpaces, nil
}

// GetDevSpaceTargetConfig retrieves the cluster configuration via graphql request
func (p *Provider) GetDevSpaceTargetConfig(devSpaceID int, target string) (*generated.DevSpaceTargetConfig, error) {
	graphQlClient := graphql.NewClient(p.Host + GraphqlEndpoint)
	req := graphql.NewRequest(`
		query($devSpaceID: Int!, $target: String!) {
			DevSpaces_by_pk(DevSpaceID: $devSpaceID) {
				deploymenttargetssBydevspaceid(where: {TargetName: {_eq: $target}}) {
					TargetName
					kubecontextsBykubecontextid {
						Namespace
						Domain
						ServiceAccountToken
						clustersByclusterid {
							CaCert
							Server
						}
					}
				}
			}
		}
	`)

	req.Var("devSpaceID", devSpaceID)
	req.Var("target", target)
	req.Header.Set("Authorization", p.Token)

	ctx := context.Background()
	response := devSpaceTargetConfigQuery{}

	// Run the graphql request
	err := graphQlClient.Run(ctx, req, &response)
	if err != nil {
		return nil, err
	}

	// Check if we got correct information
	if response.DevSpacesByPK == nil || len(response.DevSpacesByPK.DeploymenttargetssBydevspaceid) != 1 || response.DevSpacesByPK.DeploymenttargetssBydevspaceid[0].KubecontextsBykubecontextid == nil {
		return nil, fmt.Errorf("Couldn't find devSpaceID %d or target %s", devSpaceID, target)
	}

	return &generated.DevSpaceTargetConfig{
		TargetName:          response.DevSpacesByPK.DeploymenttargetssBydevspaceid[0].TargetName,
		Namespace:           response.DevSpacesByPK.DeploymenttargetssBydevspaceid[0].KubecontextsBykubecontextid.Namespace,
		ServiceAccountToken: response.DevSpacesByPK.DeploymenttargetssBydevspaceid[0].KubecontextsBykubecontextid.ServiceAccountToken,
		CaCert:              response.DevSpacesByPK.DeploymenttargetssBydevspaceid[0].KubecontextsBykubecontextid.ClustersByclusterid.CaCert,
		Server:              response.DevSpacesByPK.DeploymenttargetssBydevspaceid[0].KubecontextsBykubecontextid.ClustersByclusterid.Server,
		Domain:              response.DevSpacesByPK.DeploymenttargetssBydevspaceid[0].KubecontextsBykubecontextid.Domain,
	}, nil
}

// GetDevSpaceTargetConfigs retrieves the cluster configurations via graphql request
func (p *Provider) GetDevSpaceTargetConfigs(devSpaceID int) ([]*generated.DevSpaceTargetConfig, error) {
	graphQlClient := graphql.NewClient(p.Host + GraphqlEndpoint)
	req := graphql.NewRequest(`
		query($devSpaceID: Int!) {
			DevSpaces_by_pk(DevSpaceID: $devSpaceID) {
				deploymenttargetssBydevspaceid {
					TargetName
					kubecontextsBykubecontextid {
						Namespace
						Domain
						ServiceAccountToken
						clustersByclusterid {
							CaCert
							Server
						}
					}
				}
			}
		}
	`)

	req.Var("devSpaceID", devSpaceID)
	req.Header.Set("Authorization", p.Token)

	ctx := context.Background()
	response := devSpaceTargetConfigQuery{}

	// Run the graphql request
	err := graphQlClient.Run(ctx, req, &response)
	if err != nil {
		return nil, err
	}

	// Check if we got correct information
	if response.DevSpacesByPK == nil {
		return nil, fmt.Errorf("Error retrieving devspace targets: nil response received: %#v", response)
	}

	targets := []*generated.DevSpaceTargetConfig{}
	for _, target := range response.DevSpacesByPK.DeploymenttargetssBydevspaceid {
		if target.KubecontextsBykubecontextid != nil {
			targets = append(targets, &generated.DevSpaceTargetConfig{
				TargetName:          target.TargetName,
				Namespace:           target.KubecontextsBykubecontextid.Namespace,
				ServiceAccountToken: target.KubecontextsBykubecontextid.ServiceAccountToken,
				CaCert:              target.KubecontextsBykubecontextid.ClustersByclusterid.CaCert,
				Server:              target.KubecontextsBykubecontextid.ClustersByclusterid.Server,
				Domain:              target.KubecontextsBykubecontextid.Domain,
			})
		}
	}

	return targets, nil
}*/
