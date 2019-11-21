package client

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"

	"gotest.tools/assert"
)

func TestGetToken(t *testing.T) {
	_, err := (&client{}).GetToken()
	assert.Error(t, err, "Provider has no key specified")

	testClaim := token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
	}
	claimAsJSON, _ := json.Marshal(testClaim)
	encodedToken := "." + base64.URLEncoding.EncodeToString(claimAsJSON) + "."
	testClient := &client{
		accessKey: "someKey",
		token:     encodedToken,
	}
	token, err := testClient.GetToken()
	assert.NilError(t, err, "Error getting predefined token")
	assert.Equal(t, token, encodedToken, "Predefined valid token not returned from GetToken")

	testClient.token = ""
	_, err = testClient.GetToken()
	assert.Error(t, err, "token request: Get /auth/token?key=someKey: unsupported protocol scheme \"\"", "Wrong or no error when trying to reach an unreachable host")
}
