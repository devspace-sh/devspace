package cloud

import (
	"testing"

	client "github.com/devspace-cloud/devspace/pkg/devspace/cloud/client/testing"
	config "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/testing"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"

	"gotest.tools/assert"
)

type getClusterKeyTestCase struct {
	name string

	answers           []string
	localClusterKeys  map[int]string
	clientClusterKeys map[int]string
	clusterID         int

	expectedErr            string
	expectedKey            string
	keyExpectedInClusterID bool
}

func TestGetClusterKey(t *testing.T) {
	hash345678, err := hash.Password("345678")
	assert.NilError(t, err, "Error getting hash")
	hash456789, err := hash.Password("456789")
	assert.NilError(t, err, "Error getting hash")
	hash567890, err := hash.Password("567890")
	assert.NilError(t, err, "Error getting hash")

	testCases := []getClusterKeyTestCase{
		getClusterKeyTestCase{
			name:    "Ask for encryption key and succeed on secound try",
			answers: []string{"234567", "345678"},
			clientClusterKeys: map[int]string{
				3: hash345678,
			},
			clusterID:              3,
			expectedKey:            hash345678,
			keyExpectedInClusterID: true,
		},
		getClusterKeyTestCase{
			name:    "Get wrong clusterkey from local config, then get the right by asking",
			answers: []string{"456789"},
			localClusterKeys: map[int]string{
				1: "345678",
			},
			clientClusterKeys: map[int]string{
				5: hash456789,
			},
			clusterID:              5,
			expectedKey:            hash456789,
			keyExpectedInClusterID: true,
		},
		getClusterKeyTestCase{
			name: "Get correct clusterkey from local config",
			localClusterKeys: map[int]string{
				2: hash567890,
			},
			clientClusterKeys: map[int]string{
				6: hash567890,
			},
			clusterID:              6,
			expectedKey:            hash567890,
			keyExpectedInClusterID: true,
		},
	}

	for _, testCase := range testCases {
		logger := log.NewFakeLogger()
		for _, answer := range testCase.answers {
			logger.Survey.SetNextAnswer(answer)
		}

		if testCase.localClusterKeys == nil {
			testCase.localClusterKeys = map[int]string{}
		}

		provider := &provider{
			Provider: latest.Provider{
				ClusterKey: testCase.localClusterKeys,
			},
			log: logger,
			client: &client.CloudClient{
				ClusterKeys: testCase.clientClusterKeys,
			},
			loader: config.NewLoader(&latest.Config{}),
		}

		key, err := provider.GetClusterKey(&latest.Cluster{ClusterID: testCase.clusterID, EncryptToken: true})

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase: %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error when trying in testCase %s", testCase.name)
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
