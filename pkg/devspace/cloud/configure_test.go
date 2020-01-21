package cloud

import (
	"testing"

	"gotest.tools/assert"
)

func TestGetKubeContextNameFromSpace(t *testing.T) {
	assert.Equal(t, GetKubeContextNameFromSpace("space:Name", "provider.Name"), DevSpaceKubeContextName+"-provider-name-space-name", "Wrong KubeContextName returned")
}

func TestUpdateKubeConfig(t *testing.T) {
}
