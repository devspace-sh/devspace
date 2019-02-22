package cloud

import (
	"context"

	"github.com/machinebox/graphql"
)

// GrapqhlRequest does a new graphql request and stores the result in the response
func (p *Provider) GrapqhlRequest(request string, vars map[string]interface{}, response interface{}) error {
	graphQlClient := graphql.NewClient(p.Host + GraphqlEndpoint)
	req := graphql.NewRequest(request)

	// Set vars
	if vars != nil {
		for key, val := range vars {
			req.Var(key, val)
		}
	}

	// Set token
	req.Header.Set("Authorization", "Bearer "+p.Token)

	// Run the graphql request
	return graphQlClient.Run(context.Background(), req, response)
}
