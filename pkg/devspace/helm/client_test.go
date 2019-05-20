package helm

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	
	"gotest.tools/assert"
)

func TestNewClient(t *testing.T){
	config := createFakeConfig()
	client, err := NewClient(config, configutil.TestNamespace, log.GetInstance(), true)

	assert.Equal(t, false, err == nil, "No error when trying to create new Client without reachable ip addresses")
	assert.Equal(t, true, client == nil, "Client created despite reachable ip addresses")
}
