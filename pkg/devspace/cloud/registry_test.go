package cloud

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/client"
	fakeclient "github.com/devspace-cloud/devspace/pkg/devspace/cloud/client/testing"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	fakedocker "github.com/devspace-cloud/devspace/pkg/devspace/docker/testing"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"gotest.tools/assert"
)

type loginIntoRegistriesTestCase struct {
	name string

	cloudclient client.Client

	expectedErr string
}

func TestLoginIntoRegistries(t *testing.T) {
	claimAsJSON, _ := json.Marshal(token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
	})
	validEncodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(validEncodedClaim), "=") {
		validEncodedClaim = strings.TrimSuffix(validEncodedClaim, "=")
	}
	validToken := "." + validEncodedClaim + "."

	testCases := []loginIntoRegistriesTestCase{
		loginIntoRegistriesTestCase{
			name: "log into one registry",
			cloudclient: &fakeclient.CloudClient{
				Registries: []*latest.Registry{
					&latest.Registry{},
				},
				Token: validToken,
			},
		},
	}

	for _, testCase := range testCases {
		provider := &provider{
			client:       testCase.cloudclient,
			log:          &log.FakeLogger{},
			dockerClient: &fakedocker.FakeClient{},
		}

		err := provider.loginIntoRegistries()

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error getting Key in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from getKey in testCase %s", testCase.name)
		}
	}
}
