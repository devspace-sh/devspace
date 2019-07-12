package cloud

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/util/log"

	"gotest.tools/assert"
)

func TestGetFirstPublicRegistry(t *testing.T) {
	_, err := (&Provider{}).GetFirstPublicRegistry()
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to get first public registry without any token")
}

func TestLoginIntoRegistries(t *testing.T) {
	err := (&Provider{}).LoginIntoRegistries(&log.DiscardLogger{})
	assert.Error(t, err, "get registries: get token: Provider has no key specified", "Wrong or no error when trying log into registries without any token")
}
