package cloud

import (
	"context"
	"fmt"

	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/machinebox/graphql"
)

// DevSpaceConfig holds the information of a devspace
type DevSpaceConfig struct {
	DevSpaceID int
	Name       string
	Created    string
}

type devSpaceTargetConfigQuery struct {
	DevSpacesByPK *struct {
		DeploymenttargetssBydevspaceid []struct {
			TargetName                  string
			KubecontextsBykubecontextid *struct {
				Namespace           string
				Domain              *string
				ServiceAccountToken string
				ClustersByclusterid *struct {
					CaCert string
					Server string
				} `json:"clustersByclusterid"`
			} `json:"kubecontextsBykubecontextid"`
		} `json:"deploymenttargetssBydevspaceid"`
	} `json:"DevSpaces_by_pk"`
}

type devSpaceConfigQuery struct {
	DevSpaces []*DevSpaceConfig
}

// GetDevSpaces returns all devspaces owned by the user
func (p *Provider) GetDevSpaces() ([]*DevSpaceConfig, error) {
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
}
