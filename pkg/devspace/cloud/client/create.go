package client

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/pkg/errors"
)

// CreatePublicCluster creates a new public cluster
func (c *client) CreatePublicCluster(name, server, caCert, adminToken string) (int, error) {
	// Response struct
	response := struct {
		CreateCluster *struct {
			ClusterID int
		} `json:"manager_createCluster"`
	}{}

	// Do the request
	err := c.grapqhlRequest(`
		mutation($name:String!,$caCert:String!,$server:String!,$adminToken:String!) {
  			manager_createCluster(
				name:$name,
				caCert:$caCert,
				server:$server,
				adminToken:$adminToken
			) {
				ClusterID
			}
		}
	`, map[string]interface{}{
		"name":       name,
		"caCert":     caCert,
		"server":     server,
		"adminToken": adminToken,
	}, &response)
	if err != nil {
		return 0, err
	}

	// Check result
	if response.CreateCluster == nil {
		return 0, errors.New("Couldn't create cluster: returned answer is null")
	}

	return response.CreateCluster.ClusterID, nil
}

// CreateUserCluster creates a user cluster with the given name
func (c *client) CreateUserCluster(name, server, caCert, encryptedToken string, networkPolicyEnabled bool) (int, error) {
	// Response struct
	response := struct {
		CreateCluster *struct {
			ClusterID int
		} `json:"manager_createUserCluster"`
	}{}

	// Do the request
	err := c.grapqhlRequest(`
		mutation($name:String!,$caCert:String!,$server:String!,$encryptedToken:String!,$networkPolicyEnabled:Boolean!) {
			manager_createUserCluster(
				name:$name,
				caCert:$caCert,
				server:$server,
				encryptedToken:$encryptedToken,
				networkPolicyEnabled:$networkPolicyEnabled
			) {
				ClusterID
			}
		}
	`, map[string]interface{}{
		"name":                 name,
		"caCert":               caCert,
		"server":               server,
		"encryptedToken":       encryptedToken,
		"networkPolicyEnabled": networkPolicyEnabled,
	}, &response)
	if err != nil {
		return 0, err
	}

	// Check result
	if response.CreateCluster == nil {
		return 0, errors.New("Couldn't create cluster: returned answer is null")
	}

	return response.CreateCluster.ClusterID, nil
}

// CreateSpace creates a new space and returns the space id
func (c *client) CreateSpace(key, name string, projectID int, cluster *latest.Cluster) (int, error) {
	// Response struct
	response := struct {
		CreateSpace *struct {
			SpaceID int
		} `json:"manager_createSpace"`
	}{}

	// Do the request
	err := c.grapqhlRequest(`
		mutation($key: String, $spaceName: String!, $clusterID: Int!, $projectID: Int!) {
			manager_createSpace(key: $key, spaceName: $spaceName, clusterID: $clusterID, projectID: $projectID) {
				SpaceID
			}
		}
	`, map[string]interface{}{
		"key":       key,
		"spaceName": name,
		"projectID": projectID,
		"clusterID": cluster.ClusterID,
	}, &response)
	if err != nil {
		return 0, err
	}

	// Check result
	if response.CreateSpace == nil {
		return 0, errors.New("Couldn't create space: returned answer is null")
	}

	return response.CreateSpace.SpaceID, nil
}

// CreateProject creates a new project and returns the project id
func (c *client) CreateProject(projectName string) (int, error) {
	// Response struct
	response := struct {
		CreateProject *struct {
			ProjectID int
		} `json:"manager_createProject"`
	}{}

	// Do the request
	err := c.grapqhlRequest(`
		mutation($projectName: String!) {
			manager_createProject(projectName: $projectName) {
				ProjectID
			}
		}
	`, map[string]interface{}{
		"projectName": projectName,
	}, &response)
	if err != nil {
		return 0, err
	}

	// Check result
	if response.CreateProject == nil {
		return 0, errors.New("Couldn't create project: returned answer is null")
	}

	return response.CreateProject.ProjectID, nil
}

// CreateKubeContextDomainIngressPath creates an ingress path
func (c *client) CreateKubeContextDomainIngressPath(key string, spaceID int, ingressName, host, newPath, serviceName, servicePort string) (bool, error) {
	// Response struct
	response := struct {
		ManagerCreateIngressPath bool `json:"manager_createKubeContextDomainIngressPath"`
	}{}

	// Do the request
	err := c.grapqhlRequest(`
		mutation($spaceID: Int!, $ingressName: String!, $host: String!, $newPath: String!, $newServiceName: String!, $newServicePort: String!, $key: String) {
			manager_createKubeContextDomainIngressPath(
				spaceID: $spaceID,
				key: $key,
				ingressName: $ingressName,
				host: $host,
				newPath: $newPath,
				newServiceName: $newServiceName,
				newServicePort: $newServicePort,
			)
		}
	`, map[string]interface{}{
		"key":            key,
		"spaceID":        spaceID,
		"ingressName":    ingressName,
		"host":           host,
		"newPath":        newPath,
		"newServiceName": serviceName,
		"newServicePort": servicePort,
	}, &response)
	if err != nil {
		return false, errors.Wrap(err, "graphql create ingress path")
	}

	return response.ManagerCreateIngressPath, nil
}
