package client

import (
	"context"
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/upgrade"
	"github.com/machinebox/graphql"
	"github.com/pkg/errors"
)

// graphqlEndpoint is the endpoint where to execute graphql requests
const graphqlEndpoint = "/graphql"

// Client can communicate with the graphql-server of the cloud
type Client interface {
	CreatePublicCluster(name, server, caCert, adminToken string) (int, error)
	CreateUserCluster(name, server, caCert, encryptedToken string, networkPolicyEnabled bool) (int, error)
	CreateSpace(name, key string, projectID int, cluster *latest.Cluster) (int, error)
	CreateProject(projectName string) (int, error)

	DeleteCluster(cluster *latest.Cluster, key string, deleteServices, deleteKubeContexts bool) error
	DeleteSpace(space *latest.Space, key string) (bool, error)

	GetRegistries() ([]*latest.Registry, error)
	GetClusterByName(clusterName string) (*latest.Cluster, error)
	GetClusters() ([]*latest.Cluster, error)
	GetProjects() ([]*latest.Project, error)
	GetClusterUser(clusterID int) (*latest.ClusterUser, error)
	GetServiceAccount(space *latest.Space, key string) (*latest.ServiceAccount, error)
	GetSpaces() ([]*latest.Space, error)
	GetSpace(spaceID int) (*latest.Space, error)
	GetSpaceByName(spaceName string) (*latest.Space, error)
	VerifyKey(clusterID int, key string) (bool, error)
	Settings(encryptToken string) ([]Setting, error)

	GetToken() (string, error)

	UseDefaultClusterDomain(clusterID int, key string) (string, error)
	UpdateClusterDomain(clusterID int, domain string) error
	DeployIngressController(clusterID int, key string, useHostNetwork bool) error
	DeployAdmissionController(clusterID int, key string) error
	DeployGatekeeper(clusterID int, key string) error
	DeployGatekeeperRules(clusterID int, key string) error
	DeployCertManager(clusterID int, key string) error
	InitCore(clusterID int, key string, enablePodPolicy bool) error
	UpdateUserClusterUser(clusterUserID int, encryptedToken []byte) error
	ResumeSpace(spaceID int, key string, cluster *latest.Cluster) (bool, error)
}

// client is the default implementation of Client
type client struct {
	provider  string
	host      string
	accessKey string
	token     string

	client *graphql.Client
}

// NewClient creates a new instance of the  interface Client
func NewClient(providerName, host, accessKey, token string) Client {
	return &client{
		provider:  providerName,
		host:      host,
		accessKey: accessKey,
		token:     token,
		client:    graphql.NewClient(host + graphqlEndpoint),
	}
}

// grapqhlRequest does a new graphql request and stores the result in the response
func (c *client) grapqhlRequest(request string, vars map[string]interface{}, response interface{}) error {
	_, err := c.GetToken()
	if err != nil {
		return errors.Wrap(err, "get token")
	}

	req := graphql.NewRequest(request)

	// Set vars
	if vars != nil {
		for key, val := range vars {
			req.Var(key, val)
		}
	}

	// Set token
	req.Header.Set("Authorization", "Bearer "+c.token)

	// Run the graphql request
	err = c.client.Run(context.Background(), req, response)
	if err != nil {
		newerVersion := upgrade.NewerVersionAvailable()
		if newerVersion != "" {
			return fmt.Errorf("This error could be caused by your old DevSpace version. Please upgrade to version %s as soon as possible: \n%v", newerVersion, err)
		}

		return err
	}

	return nil
}
