package cloud

import (
	"testing"

	client "github.com/devspace-cloud/devspace/pkg/devspace/cloud/client/testing"
	config "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/testing"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"

	"gotest.tools/assert"
)

type getClusterKeyTestCase struct {
	name string

	clusterOwner   *latest.Owner
	answers        []string
	setClusterKeys map[int]string
	clusterID      int

	expectedErr            string
	expectedKey            string
	keyExpectedInClusterID bool
}

func TestGetClusterKey(t *testing.T) {
	testCases := []getClusterKeyTestCase{
		/*getClusterKeyTestCase{
			name:                   "Ask for encryption key and succeed on secound try",
			clusterOwner:           &latest.Owner{},
			answers:                []string{"234567", "345678"},
			setClusterKeys:         map[int]string{},
			expectedKey:            "d7da6caa27948d250f1ea385bf587f9d348c7334b23fa1766016b503572a73a8",
			keyExpectedInClusterID: true,
		},
		getClusterKeyTestCase{
			name:                   "Try with invalid clusterKey from map and then ask and succeed",
			clusterOwner:           &latest.Owner{},
			answers:                []string{"456789"},
			setClusterKeys:         map[int]string{2: "someKey"},
			expectedKey:            "472bbe83616e93d3c09a79103ae47d8f71e3d35a966d6e8b22f743218d04171d",
			keyExpectedInClusterID: true,
		},*/
		getClusterKeyTestCase{
			name:                   "The only clusterKey is valid and then saved to ClusterID",
			clusterOwner:           &latest.Owner{},
			setClusterKeys:         map[int]string{2: "567890"},
			expectedKey:            "567890",
			clusterID:              5,
			keyExpectedInClusterID: true,
		},
	}

	for _, testCase := range testCases {
		logger := log.NewFakeLogger()
		for _, answer := range testCase.answers {
			logger.Survey.SetNextAnswer(answer)
		}
		provider := &provider{
			Provider: latest.Provider{
				ClusterKey: testCase.setClusterKeys,
			},
			log:    logger,
			client: client.NewFakeClient(),
			loader: config.NewLoader(&latest.Config{}),
		}

		key, err := provider.GetClusterKey(&latest.Cluster{Owner: testCase.clusterOwner, ClusterID: testCase.clusterID, EncryptToken: true})

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error calling graphqlRequest in testCase: %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error when trying to do a graphql request in testCase %s", testCase.name)
		}
		assert.Equal(t, testCase.expectedKey, key, "Wrong key returned in testCase %s", testCase.name)

		if testCase.keyExpectedInClusterID {
			assert.Equal(t, testCase.expectedKey, provider.ClusterKey[testCase.clusterID], "Wrong key returned in clusterKey with clusterID %s", testCase.name)
		} else {
			_, ok := provider.ClusterKey[testCase.clusterID]
			assert.Equal(t, false, ok, "ClusterKey with clusterID unexpectedly set. TestCase: %s", testCase.name)
		}
	}

}
