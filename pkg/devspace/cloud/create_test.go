package cloud

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"gotest.tools/assert"
)

func TestCreateUserCluster(t *testing.T) {
	provider := &Provider{latest.Provider{}, log.GetInstance()}
	_, err := provider.CreateUserCluster("", "", "", "", false)
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to create a usercluster without a token")
}

func TestCreateSpace(t *testing.T) {
	provider := &Provider{latest.Provider{}, log.GetInstance()}
	_, err := provider.CreateSpace("", 0, &latest.Cluster{})
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to create a space without a token")
}

func TestCreateProject(t *testing.T) {
	provider := &Provider{latest.Provider{}, log.GetInstance()}
	_, err := provider.CreateProject("")
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to create a project without a token")
}
