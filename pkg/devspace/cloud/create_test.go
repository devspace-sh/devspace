package cloud

import (
	"testing"

	"gotest.tools/assert"
)

func TestCreateUserCluster(t *testing.T) {
	provider := &Provider{}
	_, err := provider.CreateUserCluster("", "", "", "", false)
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to create a usercluster without a token")
}

func TestCreateSpace(t *testing.T) {
	provider := &Provider{}
	_, err := provider.CreateSpace("", 0, &Cluster{})
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to create a space without a token")
}

func TestCreateProject(t *testing.T) {
	provider := &Provider{}
	_, err := provider.CreateProject("")
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to create a project without a token")
}
