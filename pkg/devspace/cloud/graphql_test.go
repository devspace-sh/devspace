package cloud

import (
	"testing"

	"gotest.tools/assert"
)

func TestGrapqhlRequest(t *testing.T) {
	provider := &Provider{}
	err := provider.GrapqhlRequest("", map[string]interface{}{"hello": "world"}, nil)
	assert.Error(t, err, "get token: Provider has no key specified", "Wrong or no error when trying to do a graphql request without a token")
}
