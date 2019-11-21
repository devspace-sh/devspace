package cloud

/*import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	homedir "github.com/mitchellh/go-homedir"

	"gotest.tools/assert"
)

type getClusterKeyTestCase struct {
	name string

	keyVerifiedInResponse bool
	clusterOwner          *latest.Owner
	setAnswers            []string
	setClusterKeys        map[int]string
	clusterID             int
	graphQLClient         *fakeGraphQLClient

	expectedErr            string
	expectedKey            string
	keyExpectedInClusterID bool
}

func TestGetClusterKey(t *testing.T) {
	testCases := []getClusterKeyTestCase{
		getClusterKeyTestCase{
			name:         "Ask for encryption key and succeed on secound try",
			clusterOwner: &latest.Owner{},
			setAnswers:   []string{"234567", "345678"},
			graphQLClient: &fakeGraphQLClient{
				responsesAsJSON: []string{"{\"manager_verifyUserClusterKey\":false}", "{\"manager_verifyUserClusterKey\":true}"},
			},
			setClusterKeys:         map[int]string{},
			expectedKey:            "d7da6caa27948d250f1ea385bf587f9d348c7334b23fa1766016b503572a73a8",
			keyExpectedInClusterID: true,
		},
		getClusterKeyTestCase{
			name:           "Try with invalid clusterKey from map and then ask and succeed",
			clusterOwner:   &latest.Owner{},
			setAnswers:     []string{"456789"},
			setClusterKeys: map[int]string{2: "someKey"},
			graphQLClient: &fakeGraphQLClient{
				responsesAsJSON: []string{"{\"manager_verifyUserClusterKey\":false}", "{\"manager_verifyUserClusterKey\":true}"},
			},
			expectedKey:            "472bbe83616e93d3c09a79103ae47d8f71e3d35a966d6e8b22f743218d04171d",
			keyExpectedInClusterID: true,
		},
		getClusterKeyTestCase{
			name:           "Only clusterKey is valid and then saved to ClusterID",
			clusterOwner:   &latest.Owner{},
			setClusterKeys: map[int]string{2: "567890"},
			graphQLClient: &fakeGraphQLClient{
				responsesAsJSON: []string{"{\"manager_verifyUserClusterKey\":true}"},
			},
			expectedKey:            "567890",
			clusterID:              5,
			keyExpectedInClusterID: true,
		},
	}

	//Create tempDir and go into it
	dir, err := ioutil.TempDir("", "testDir")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	//Make backup file
	homedir, err := homedir.Dir()
	assert.NilError(t, err, "Error finding out home directory")
	providerFile := filepath.Join(homedir, cloudconfig.DevSpaceProvidersConfigPath)
	err = fsutil.Copy(providerFile, "providersBackup", true)
	didConfigExist := !os.IsNotExist(err)

	// Delete temp folder after test
	defer func() {
		if didConfigExist {
			err = fsutil.Copy("providersBackup", providerFile, true)
		} else {
			err = os.Remove(providerFile)
		}
		assert.NilError(t, err, "Error restoring config")

		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	for _, testCase := range testCases {
		provider := Provider{
			latest.Provider{
				ClusterKey: testCase.setClusterKeys,
			},
			log.GetInstance(),
		}

		if testCase.setAnswers == nil {
			testCase.setAnswers = []string{}
		}
		for _, answer := range testCase.setAnswers {
			survey.SetNextAnswer(answer)
		}
		if testCase.graphQLClient != nil {
			DefaultGraphqlClient = testCase.graphQLClient
		}

		key, err := provider.getClusterKey(&latest.Cluster{Owner: testCase.clusterOwner, ClusterID: testCase.clusterID, EncryptToken: true})
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error calling graphqlRequest in testCase: %s", testCase.name)
			assert.Equal(t, testCase.expectedKey, key, "Wrong key returned in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error when trying to do a graphql request in testCase %s", testCase.name)
		}
		if testCase.keyExpectedInClusterID {
			assert.Equal(t, testCase.expectedKey, provider.ClusterKey[testCase.clusterID], "Wrong key returned in clusterKey with clusterID %s", testCase.name)
		} else {
			_, ok := provider.ClusterKey[testCase.clusterID]
			assert.Equal(t, false, ok, "ClusterKey with clusterID unexpectedly set. TestCase: %s", testCase.name)
		}

		DefaultGraphqlClient = &GraphqlClient{}
	}
}
*/