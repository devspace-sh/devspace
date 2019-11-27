package testing

import (
	cloudclient "github.com/devspace-cloud/devspace/pkg/devspace/cloud/client"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
)

type cloudClient struct{}

// NewFakeClient creates a test instance of the cloud-client interface
func NewFakeClient() cloudclient.Client {
	return &cloudClient{}
}

func (c *cloudClient) CreatePublicCluster(name, server, caCert, adminToken string) (int, error) {
	return 1, nil
}

func (c *cloudClient) CreateUserCluster(name, server, caCert, encryptedToken string, networkPolicyEnabled bool) (int, error) {
	return 1, nil
}

func (c *cloudClient) CreateSpace(name, key string, projectID int, cluster *latest.Cluster) (int, error) {
	return 1, nil
}

func (c *cloudClient) CreateProject(projectName string) (int, error) {
	return 1, nil
}

func (c *cloudClient) DeleteCluster(cluster *latest.Cluster, key string, deleteServices, deleteKubeContexts bool) error {
	return nil
}

func (c *cloudClient) DeleteSpace(space *latest.Space, key string) (bool, error) {
	return true, nil
}

func (c *cloudClient) GetRegistries() ([]*latest.Registry, error) {
	return []*latest.Registry{}, nil
}

func (c *cloudClient) GetClusterByName(clusterName string) (*latest.Cluster, error) {
	return &latest.Cluster{}, nil
}

func (c *cloudClient) GetClusters() ([]*latest.Cluster, error) {
	return []*latest.Cluster{}, nil
}

func (c *cloudClient) GetProjects() ([]*latest.Project, error) {
	return []*latest.Project{}, nil
}

func (c *cloudClient) GetClusterUser(clusterID int) (*latest.ClusterUser, error) {
	return &latest.ClusterUser{}, nil
}

func (c *cloudClient) GetServiceAccount(space *latest.Space, key string) (*latest.ServiceAccount, error) {
	return &latest.ServiceAccount{}, nil
}

func (c *cloudClient) GetSpaces() ([]*latest.Space, error) {
	return []*latest.Space{}, nil
}

func (c *cloudClient) GetSpace(spaceID int) (*latest.Space, error) {
	return &latest.Space{}, nil
}

func (c *cloudClient) GetSpaceByName(spaceName string) (*latest.Space, error) {
	return &latest.Space{}, nil
}

func (c *cloudClient) VerifyKey(clusterID int, key string) (bool, error) {
	return true, nil
}

func (c *cloudClient) Settings(encryptToken string) ([]cloudclient.Setting, error) {
	return []cloudclient.Setting{}, nil
}

func (c *cloudClient) GetToken() (string, error) {
	return "", nil
}

func (c *cloudClient) UseDefaultClusterDomain(clusterID int, key string) (string, error) {
	return "", nil
}

func (c *cloudClient) UpdateClusterDomain(clusterID int, domain string) error {
	return nil
}

func (c *cloudClient) DeployIngressController(clusterID int, key string, useHostNetwork bool) error {
	return nil
}

func (c *cloudClient) DeployAdmissionController(clusterID int, key string) error {
	return nil
}
func (c *cloudClient) DeployGatekeeper(clusterID int, key string) error {
	return nil
}

func (c *cloudClient) DeployGatekeeperRules(clusterID int, key string) error {
	return nil
}

func (c *cloudClient) DeployCertManager(clusterID int, key string) error {
	return nil
}

func (c *cloudClient) InitCore(clusterID int, key string, enablePodPolicy bool) error {
	return nil
}

func (c *cloudClient) UpdateUserClusterUser(clusterUserID int, encryptedToken []byte) error {
	return nil
}

func (c *cloudClient) ResumeSpace(spaceID int, key string, cluster *latest.Cluster) (bool, error) {
	return true, nil
}
