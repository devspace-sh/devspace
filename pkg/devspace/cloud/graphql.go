package cloud

import (
	"context"

	"github.com/devspace-cloud/devspace/pkg/devspace/upgrade"
	"github.com/machinebox/graphql"
	"github.com/pkg/errors"
)

//DefaultGraphqlClient is the default client for graphQL requests. It is overwritable
var DefaultGraphqlClient GraphqlClientInterface = &GraphqlClient{}

//GraphqlClientInterface can communicate with a graphQL server
type GraphqlClientInterface interface {
	GrapqhlRequest(p *Provider, request string, vars map[string]interface{}, response interface{}) error
}

//GraphqlClient is the default implementation of the GraphqlClientInterface
type GraphqlClient struct{}

// GrapqhlRequest does a new graphql request and stores the result in the response
func (g *GraphqlClient) GrapqhlRequest(p *Provider, request string, vars map[string]interface{}, response interface{}) error {
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
	err = graphQlClient.Run(context.Background(), req, response)
	if err != nil {
		newerVersion := upgrade.NewerVersionAvailable()
		if newerVersion != "" {
			p.Log.Warnf("This error could be caused by your old DevSpace version. Please upgrade to version %s as soon as possible", newerVersion)
		}

		return err
	}

	return nil
}

// GrapqhlRequest does a new graphql request and stores the result in the response
func (p *Provider) GrapqhlRequest(request string, vars map[string]interface{}, response interface{}) error {
	return DefaultGraphqlClient.GrapqhlRequest(p, request, vars, response)
}
