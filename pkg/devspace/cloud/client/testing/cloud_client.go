package testing

import (
	"errors"
	"strings"

	cloudclient "github.com/devspace-cloud/devspace/pkg/devspace/cloud/client"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
)

// ExtendedCluster is an extended latest.Cluster with more required fields
type ExtendedCluster struct {
	latest.Cluster
	Domain   string
	Deployed []string
}

// CloudClient is a fake version of the cloud client
type CloudClient struct {
	Spaces      []*latest.Space
	Registries  []*latest.Registry
	Clusters    []*ExtendedCluster
	ClusterKeys map[int]string
	Projects    []*latest.Project
	SettingsArr []cloudclient.Setting
	Token       string
}

// CreatePublicCluster is a fake implementation for that function
func (c *CloudClient) CreatePublicCluster(name, server, caCert, adminToken string) (int, error) {
	clusterID := len(c.Clusters)
	c.Clusters = append(c.Clusters, &ExtendedCluster{
		Cluster: latest.Cluster{
			ClusterID: clusterID,
			Name:      name,
			Server:    &server,
		},
	})
	return clusterID, nil
}

// CreateUserCluster is a fake implementation for that function
func (c *CloudClient) CreateUserCluster(name, server, caCert, encryptedToken string, networkPolicyEnabled bool) (int, error) {
	clusterID := len(c.Clusters)
	c.Clusters = append(c.Clusters, &ExtendedCluster{
		Cluster: latest.Cluster{
			ClusterID:    clusterID,
			Name:         name,
			Server:       &server,
			EncryptToken: true,
		},
	})
	return clusterID, nil
}

// CreateSpace is a fake implementation for that function
func (c *CloudClient) CreateSpace(name, key string, projectID int, cluster *latest.Cluster) (int, error) {
	spaceID := len(c.Spaces)
	c.Spaces = append(c.Spaces, &latest.Space{
		SpaceID: spaceID,
		Name:    name,
		Cluster: cluster,
	})
	return spaceID, nil
}

// CreateProject is a fake implementation for that function
func (c *CloudClient) CreateProject(projectName string) (int, error) {
	projectID := len(c.Projects)
	c.Projects = append(c.Projects, &latest.Project{
		ProjectID: projectID,
		Name:      projectName,
	})
	return projectID, nil
}

// DeleteCluster is a fake implementation for that function
func (c *CloudClient) DeleteCluster(cluster *latest.Cluster, key string, deleteServices, deleteKubeContexts bool) error {
	clusterID := cluster.ClusterID

	if c.ClusterKeys[clusterID] != key {
		return errors.New("Wrong key")
	}

	for i, cluster := range c.Clusters {
		if cluster.ClusterID == clusterID {
			c.Clusters[i] = c.Clusters[len(c.Clusters)-1]
			c.Clusters = c.Clusters[:len(c.Clusters)-1]
			delete(c.ClusterKeys, clusterID)
			return nil
		}
	}

	return errors.New("Cluster not found")
}

// DeleteSpace is a fake implementation for that function
func (c *CloudClient) DeleteSpace(space *latest.Space, key string) (bool, error) {
	spaceID := space.SpaceID
	for i, space := range c.Spaces {
		if space.SpaceID == spaceID {
			c.Spaces[i] = c.Spaces[len(c.Spaces)-1]
			c.Spaces = c.Spaces[:len(c.Spaces)-1]
			return true, nil
		}
	}

	return false, nil
}

// GetRegistries is a fake implementation for that function
func (c *CloudClient) GetRegistries() ([]*latest.Registry, error) {
	return c.Registries, nil
}

// GetClusterByName is a fake implementation for that function
func (c *CloudClient) GetClusterByName(clusterName string) (*latest.Cluster, error) {
	for _, cluster := range c.Clusters {
		if cluster.Name == clusterName {
			return &cluster.Cluster, nil
		}
	}

	return nil, errors.New("Cluster not found")
}

// GetClusters is a fake implementation for that function
func (c *CloudClient) GetClusters() ([]*latest.Cluster, error) {
	clusters := []*latest.Cluster{}
	for _, cluster := range c.Clusters {
		clusters = append(clusters, &cluster.Cluster)
	}
	return clusters, nil
}

// GetProjects is a fake implementation for that function
func (c *CloudClient) GetProjects() ([]*latest.Project, error) {
	return c.Projects, nil
}

// GetClusterUser is a fake implementation for that function
func (c *CloudClient) GetClusterUser(clusterID int) (*latest.ClusterUser, error) {
	for _, cluster := range c.Clusters {
		if cluster.ClusterID == clusterID {
			if cluster.Owner == nil {
				cluster.Owner = &latest.Owner{}
			}
			return &latest.ClusterUser{
				ClusterUserID: cluster.Owner.OwnerID,
				AccountID:     cluster.Owner.OwnerID,
				ClusterID:     clusterID,
				IsAdmin:       strings.Contains(cluster.Owner.Name, "admin"),
			}, nil
		}
	}
	return nil, errors.New("Cluster not found")
}

func (c *CloudClient) GetServiceAccount(space *latest.Space, key string) (*latest.ServiceAccount, error) {
	return &latest.ServiceAccount{}, nil
}

// GetSpaces is a fake implementation for that function
func (c *CloudClient) GetSpaces() ([]*latest.Space, error) {
	return c.Spaces, nil
}

// GetSpace is a fake implementation for that function
func (c *CloudClient) GetSpace(spaceID int) (*latest.Space, error) {
	for _, space := range c.Spaces {
		if space.SpaceID == spaceID {
			return space, nil
		}
	}

	return nil, errors.New("Space not found")
}

// GetSpaceByName is a fake implementation for that function
func (c *CloudClient) GetSpaceByName(spaceName string) (*latest.Space, error) {
	for _, space := range c.Spaces {
		if space.Name == spaceName {
			return space, nil
		}
	}

	return nil, errors.New("Space not found")
}

// VerifyKey is a fake implementation for that function
func (c *CloudClient) VerifyKey(clusterID int, key string) (bool, error) {
	return c.ClusterKeys[clusterID] == key, nil
}

// Settings is a fake implementation for that function
func (c *CloudClient) Settings(encryptToken string) ([]cloudclient.Setting, error) {
	return c.SettingsArr, nil
}

// GetToken is a fake implementation for that function
func (c *CloudClient) GetToken() (string, error) {
	return c.Token, nil
}

// UseDefaultClusterDomain is a fake implementation for that function
func (c *CloudClient) UseDefaultClusterDomain(clusterID int, key string) (string, error) {
	if c.ClusterKeys[clusterID] != key {
		return "", errors.New("Wrong key")
	}

	for _, cluster := range c.Clusters {
		if cluster.ClusterID == clusterID {
			cluster.Domain = "default"
			return "default", nil
		}
	}
	return "", errors.New("Cluster not found")
}

// UpdateClusterDomain is a fake implementation for that function
func (c *CloudClient) UpdateClusterDomain(clusterID int, domain string) error {
	for _, cluster := range c.Clusters {
		if cluster.ClusterID == clusterID {
			cluster.Domain = domain
			return nil
		}
	}
	return errors.New("Cluster not found")
}

// DeployIngressController is a fake implementation for that function
func (c *CloudClient) DeployIngressController(clusterID int, key string, useHostNetwork bool) error {
	if c.ClusterKeys[clusterID] != key {
		return errors.New("Wrong key " + key)
	}

	for _, cluster := range c.Clusters {
		if cluster.ClusterID == clusterID {
			cluster.Deployed = append(cluster.Deployed, "IngressController")
			if useHostNetwork {
				cluster.Cluster.Server = ptr.String("HostNetwork")
			}
			return nil
		}
	}

	return errors.New("Cluster not found")
}

// DeployAdmissionController is a fake implementation for that function
func (c *CloudClient) DeployAdmissionController(clusterID int, key string) error {
	if c.ClusterKeys[clusterID] != key {
		return errors.New("Wrong key")
	}

	for _, cluster := range c.Clusters {
		if cluster.ClusterID == clusterID {
			cluster.Deployed = append(cluster.Deployed, "AdmissionController")
			return nil
		}
	}

	return errors.New("Cluster not found")
}

// DeployGatekeeper is a fake implementation for that function
func (c *CloudClient) DeployGatekeeper(clusterID int, key string) error {
	if c.ClusterKeys[clusterID] != key {
		return errors.New("Wrong key")
	}

	for _, cluster := range c.Clusters {
		if cluster.ClusterID == clusterID {
			cluster.Deployed = append(cluster.Deployed, "Gatekeeper")
			return nil
		}
	}

	return errors.New("Cluster not found")
}

// DeployGatekeeperRules is a fake implementation for that function
func (c *CloudClient) DeployGatekeeperRules(clusterID int, key string) error {
	if c.ClusterKeys[clusterID] != key {
		return errors.New("Wrong key")
	}

	for _, cluster := range c.Clusters {
		if cluster.ClusterID == clusterID {
			cluster.Deployed = append(cluster.Deployed, "GatekeeperRules")
			return nil
		}
	}

	return errors.New("Cluster not found")
}

// DeployCertManager is a fake implementation for that function
func (c *CloudClient) DeployCertManager(clusterID int, key string) error {
	if c.ClusterKeys[clusterID] != key {
		return errors.New("Wrong key")
	}

	for _, cluster := range c.Clusters {
		if cluster.ClusterID == clusterID {
			cluster.Deployed = append(cluster.Deployed, "CertManager")
			return nil
		}
	}

	return errors.New("Cluster not found")
}

func (c *CloudClient) InitCore(clusterID int, key string, enablePodPolicy bool) error {
	return nil
}

func (c *CloudClient) UpdateUserClusterUser(clusterUserID int, encryptedToken []byte) error {
	return nil
}

func (c *CloudClient) ResumeSpace(spaceID int, key string, cluster *latest.Cluster) (bool, error) {
	return true, nil
}
