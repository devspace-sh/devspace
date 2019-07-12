package configutil

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"gotest.tools/assert"
)

func TestSetFakeConfig(t *testing.T) {
	// Create fake devspace config
	testConfig := &latest.Config{
		Deployments: &[]*latest.DeploymentConfig{
			&latest.DeploymentConfig{
				Name: ptr.String("test-deployment"),
			},
		},
	}
	SetFakeConfig(testConfig)

	assert.Equal(t, len(*config.Deployments), 1, "Config not set")
	assert.Equal(t, *((*config.Deployments)[0].Name), "test-deployment", "Config not set")

	SetFakeConfig(&latest.Config{})
	assert.Equal(t, len(*config.Deployments), 0, "Config not set")
}
