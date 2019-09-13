package cloud

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/pkg/errors"
	"gotest.tools/assert"
)

type graphqlRequestTestCase struct {
	name string

	providerKey   string
	providerToken string
	vars          map[string]interface{}

	expectedErr string
}

func TestGrapqhlRequest(t *testing.T) {
	testClaim := token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
	}
	claimAsJSON, _ := json.Marshal(testClaim)
	encodedToken := "." + base64.URLEncoding.EncodeToString(claimAsJSON) + "."
	testCases := []graphqlRequestTestCase{
		graphqlRequestTestCase{
			name:        "Test with empty provider",
			expectedErr: "get token: Provider has no key specified",
		},
		graphqlRequestTestCase{
			name:          "Test with valid token",
			providerKey:   "a",
			providerToken: encodedToken,
			vars: map[string]interface{}{
				"hello": "world",
			},
			expectedErr: "Post /graphql: unsupported protocol scheme \"\"",
		},
	}

	for _, testCase := range testCases {
		provider := &Provider{
			latest.Provider{
				Key:   testCase.providerKey,
				Token: testCase.providerToken,
			},
		}
		err := provider.GrapqhlRequest("", testCase.vars, nil)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error calling graphqlRequest in testCase: %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error when trying to do a graphql request in testCase %s", testCase.name)
		}
	}
}

type fakeGraphQLClient struct {
	responsesAsJSON []string
	errorReturn     error
}

func (fake *fakeGraphQLClient) GrapqhlRequest(p *Provider, request string, vars map[string]interface{}, response interface{}) error {
	err := json.Unmarshal([]byte(fake.responsesAsJSON[0]), response)
	fake.responsesAsJSON = fake.responsesAsJSON[1:]
	if err != nil {
		return errors.Errorf("Error parsing given json: %v", err)
	}
	return fake.errorReturn
}
