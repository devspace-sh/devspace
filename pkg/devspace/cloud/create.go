package cloud

import (
	"context"
	"errors"
	"fmt"

	"github.com/machinebox/graphql"
)

type managerCreateDevSpaceMutation struct {
	ManagerCreateDevSpace *struct {
		DevSpaceID int
	} `json:"manager_createDevSpace"`
}

type managerCreateDevSpaceTargetMutation struct {
	ManagerCreateDevSpaceTarget *struct {
		KubeContextID int
	} `json:"manager_createDevSpaceTarget"`
}

// CreateDevSpace creates a new devspace remotely
func (p *Provider) CreateDevSpace(name string) (int, error) {
	graphQlClient := graphql.NewClient(p.Host + GraphqlEndpoint)
	req := graphql.NewRequest(`
		mutation($devSpaceName: String!) {
			manager_createDevSpace(devSpaceName: $devSpaceName) {
				DevSpaceID
			}
		}
	`)

	req.Var("devSpaceName", name)
	req.Header.Set("Authorization", p.Token)

	ctx := context.Background()
	response := managerCreateDevSpaceMutation{}

	// Run the graphql request
	err := graphQlClient.Run(ctx, req, &response)
	if err != nil {
		return 0, err
	}

	if response.ManagerCreateDevSpace == nil {
		return 0, errors.New("Couldn't create devspace: returned devSpaceID is null")
	}

	return response.ManagerCreateDevSpace.DevSpaceID, nil
}

// CreateDevSpaceTarget creates a new target for an existing devspace
func (p *Provider) CreateDevSpaceTarget(devSpaceID int, target string) error {
	graphQlClient := graphql.NewClient(p.Host + GraphqlEndpoint)
	req := graphql.NewRequest(`
		mutation($devSpaceID: Int!, $target: String!) {
			manager_createDevSpaceTarget(devSpaceID: $devSpaceID, target: $target) {
				KubeContextID
			}
		}
	`)

	req.Var("devSpaceID", devSpaceID)
	req.Var("target", target)
	req.Header.Set("Authorization", p.Token)

	ctx := context.Background()
	response := managerCreateDevSpaceTargetMutation{}

	// Run the graphql request
	err := graphQlClient.Run(ctx, req, &response)
	if err != nil {
		return err
	}

	if response.ManagerCreateDevSpaceTarget == nil {
		return fmt.Errorf("Couldn't create devspace target %s: returned kubecontext is null", target)
	}

	return nil
}
