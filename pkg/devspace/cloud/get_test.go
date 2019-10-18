package cloud

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"gotest.tools/assert"
)

func TestGetRegistries(t *testing.T) {
	provider := &Provider{latest.Provider{}, log.GetInstance()}
	_, err := provider.GetRegistries()
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to get registries without a token")
}

func TestGetClusterByName(t *testing.T) {
	provider := &Provider{latest.Provider{}, log.GetInstance()}
	_, err := provider.GetClusterByName("")
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to get a cluster without a token")
}

func TestGetClusters(t *testing.T) {
	provider := &Provider{latest.Provider{}, log.GetInstance()}
	_, err := provider.GetClusters()
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to get clusters without a token")
}

func TestGetProjects(t *testing.T) {
	provider := &Provider{latest.Provider{}, log.GetInstance()}
	_, err := provider.GetProjects()
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to get projects without a token")
}

func TestGetClusterUser(t *testing.T) {
	provider := &Provider{latest.Provider{}, log.GetInstance()}
	_, err := provider.GetClusterUser(0)
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to get a cluster user without a token")
}

func TestGetServiceAccount(t *testing.T) {
	provider := &Provider{latest.Provider{}, log.GetInstance()}
	_, err := provider.GetServiceAccount(&latest.Space{Cluster: &latest.Cluster{}})
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to get a service account without a token")
}

func TestGetSpaces(t *testing.T) {
	provider := &Provider{latest.Provider{}, log.GetInstance()}
	_, err := provider.GetSpaces()
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to get spaces without a token")
}

func TestGetSpace(t *testing.T) {
	provider := &Provider{latest.Provider{}, log.GetInstance()}
	_, err := provider.GetSpace(0)
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to get a space without a token")
}

func TestGetSpaceByName(t *testing.T) {
	provider := &Provider{latest.Provider{}, log.GetInstance()}
	_, err := provider.GetSpaceByName(":")
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to get a space without a token")
}
