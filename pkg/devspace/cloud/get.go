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
}

type devSpaceTargetConfigQuery struct {
	DevSpacesByPK *struct {
		DeploymenttargetssBydevspaceid []struct {
			KubecontextsBykubecontextid *struct {
				Namespace           string
				Domain              *string
				ServiceAccountToken string
				ClustersByclusterid *struct {
					CaCert string
					Server string
				} `json: "clustersByclusterid"`
			} `json: "kubecontextsBykubecontextid"`
		} `json: "deploymenttargetssBydevspaceid"`
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
		Namespace:           response.DevSpacesByPK.DeploymenttargetssBydevspaceid[0].KubecontextsBykubecontextid.Namespace,
		ServiceAccountToken: response.DevSpacesByPK.DeploymenttargetssBydevspaceid[0].KubecontextsBykubecontextid.ServiceAccountToken,
		CaCert:              response.DevSpacesByPK.DeploymenttargetssBydevspaceid[0].KubecontextsBykubecontextid.ClustersByclusterid.CaCert,
		Server:              response.DevSpacesByPK.DeploymenttargetssBydevspaceid[0].KubecontextsBykubecontextid.ClustersByclusterid.Server,
		Domain:              response.DevSpacesByPK.DeploymenttargetssBydevspaceid[0].KubecontextsBykubecontextid.Domain,
	}, nil
}
