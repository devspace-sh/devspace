package cloud

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"gotest.tools/assert"
)

type getKeyTestCase struct {
	name string

	givenKeys          map[int]string
	forceQuestionParam bool
	answers            []string

	expectedErr string
	expectedKey string
}

func TestGetKey(t *testing.T) {
	testCases := []getKeyTestCase{
		getKeyTestCase{
			name:               "One key, no question",
			givenKeys:          map[int]string{5: "onlyKey"},
			forceQuestionParam: false,
			expectedKey:        "onlyKey",
		},
		getKeyTestCase{
			name:               "Key from question",
			forceQuestionParam: true,
			answers:            []string{"firstKey", "secondKey", "sameKey", "sameKey"},
			expectedKey:        "716fb307cf5cc64f34acfe748560a1a268d6e1a47d56ff1fc64eb549bcecd3f1",
		},
	}

	for _, testCase := range testCases {
		logger := log.NewFakeLogger()
		for _, answer := range testCase.answers {
			logger.Survey.SetNextAnswer(answer)
		}
		provider := &provider{
			Provider: latest.Provider{
				ClusterKey: testCase.givenKeys,
			},
			log: logger,
		}

		returnedKey, err := provider.getKey(testCase.forceQuestionParam)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error getting Key in testCase %s", testCase.name)
			assert.Equal(t, returnedKey, testCase.expectedKey, "Wrong key returned in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from getKey in testCase %s", testCase.name)
		}
	}
}

type getClusternameTestCase struct {
	name string

	clusterName string
	answers     []string

	expectedErr         string
	expectedClustername string
}

func TestGetClustername(t *testing.T) {
	testCases := []getClusternameTestCase{
		getClusternameTestCase{
			name:        "Invalid clustername",
			clusterName: "%",
			expectedErr: "Cluster name % can only contain letters, numbers and dashes (-)",
		},
		getClusternameTestCase{
			name:                "Valid clustername",
			clusterName:         "valid-name-1",
			expectedClustername: "valid-name-1",
		},
		getClusternameTestCase{
			name:                "Clustername from question",
			answers:             []string{"()", "valid-name-2"},
			expectedClustername: "valid-name-2",
		},
	}

	for _, testCase := range testCases {
		logger := log.NewFakeLogger()
		for _, answer := range testCase.answers {
			logger.Survey.SetNextAnswer(answer)
		}
		provider := &provider{
			log: logger,
		}

		clusterName, err := provider.getClusterName(testCase.clusterName)

		assert.Equal(t, clusterName, testCase.expectedClustername, "Wrong key returned in testCase %s", testCase.name)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
	}
}

// Kubectl fake client still missing
/*type checkResourcesTestCase struct {
	name         string
	provider     *provider
	createdNodes []*k8sv1.Node

	expectedErr string
}

func TestCheckResources(t *testing.T) {
	testCases := []checkResourcesTestCase{}

	for _, testCase := range testCases {
		kubeClient := fake.NewSimpleClientset()
		for _, node := range testCase.createdNodes {
			kubeClient.CoreV1().Nodes().Create(node)
		}

		_, err := testCase.provider.checkResources(kubeClient)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error checking resources in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error from checking resources in testCase %s", testCase.name)
		}
	}
}*/
