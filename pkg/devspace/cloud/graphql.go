package cloud

import (
	"context"

	"github.com/machinebox/graphql"
	"github.com/pkg/errors"
)

var defaultGraphlClient graphqlClientInterface = &graphlClient{}

type graphqlClientInterface interface {
	GrapqhlRequest(p *Provider, request string, vars map[string]interface{}, response interface{}) error
}

type graphlClient struct{}

// GrapqhlRequest does a new graphql request and stores the result in the response
func (g *graphlClient) GrapqhlRequest(p *Provider, request string, vars map[string]interface{}, response interface{}) error {
	graphQlClient := graphql.NewClient(p.Host + GraphqlEndpoint)
	req := graphql.NewRequest(request)

	// Set vars
	if vars != nil {
		for key, val := range vars {
			req.Var(key, val)
		}
	}

	// Get token
	token, err := p.GetToken()
	if err != nil {
		return errors.Wrap(err, "get token")
	}

	// Set token
	req.Header.Set("Authorization", "Bearer "+token)

	// Run the graphql request
	return graphQlClient.Run(context.Background(), req, response)
}

// GrapqhlRequest does a new graphql request and stores the result in the response
func (p *Provider) GrapqhlRequest(request string, vars map[string]interface{}, response interface{}) error {
	return defaultGraphlClient.GrapqhlRequest(p, request, vars, response)
}
