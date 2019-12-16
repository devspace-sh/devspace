package remove

/*import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	homedir "github.com/mitchellh/go-homedir"

	"gotest.tools/assert"
)

type removeSpaceTestCase struct {
	name string

	fakeConfig *latest.Config

	args             []string
	answers          []string
	graphQLResponses []interface{}
	spaceID          string
	provider         string
	all              bool
	providerList     []*cloudlatest.Provider

	expectedErr string
}

func TestRunRemoveSpace(t *testing.T) {
	claimAsJSON, _ := json.Marshal(token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
	})
	validEncodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(validEncodedClaim), "=") {
		validEncodedClaim = strings.TrimSuffix(validEncodedClaim, "=")
	}

	testCases := []removeSpaceTestCase{
		removeSpaceTestCase{
			name:     "Delete all one spaces",
			provider: "myProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name:  "myProvider",
					Key:   "someKey",
					Token: "." + validEncodedClaim + ".",
				},
			},
			all: true,
			graphQLResponses: []interface{}{
				struct {
					Spaces []interface{} `json:"space"`
				}{
					Spaces: []interface{}{
						struct {
							Owner   struct{} `json:"account"`
							Context struct {
								Cluster struct{} `json:"cluster"`
							} `json:"kube_context"`
						}{},
					},
				},
				struct {
					ManagerDeleteSpace bool `json:"manager_deleteSpace"`
				}{
					ManagerDeleteSpace: true,
				},
			},
		},
		removeSpaceTestCase{
			name:     "Delete one space successfully",
			provider: "myProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name:  "myProvider",
					Key:   "someKey",
					Token: "." + validEncodedClaim + ".",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Spaces []interface{} `json:"space"`
				}{
					Spaces: []interface{}{
						struct {
							Owner   struct{} `json:"account"`
							Context struct {
								Cluster struct{} `json:"cluster"`
							} `json:"kube_context"`
						}{},
					},
				},
				struct {
					ManagerDeleteSpace bool `json:"manager_deleteSpace"`
				}{
					ManagerDeleteSpace: true,
				},
			},
			args:       []string{"a:b"},
			fakeConfig: &latest.Config{},
		},
	}

	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testRunRemoveSpace(t, testCase)
	}
}

func testRunRemoveSpace(t *testing.T, testCase removeSpaceTestCase) {
	dir, err := ioutil.TempDir("", "test")
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

	homedir, err := homedir.Dir()
	assert.NilError(t, err, "Error getting homedir in testCase %s", testCase.name)
	relDir, err := filepath.Rel(homedir, dir)
	assert.NilError(t, err, "Error getting relative dir path in testCase %s", testCase.name)
	cloudconfig.DevSpaceProvidersConfigPath = filepath.Join(relDir, "Doesn'tExist")
	cloudconfig.LegacyDevSpaceCloudConfigPath = filepath.Join(relDir, "Doesn'tExist")

	providerConfig, err := cloudconfig.Load()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	cloudpkg.DefaultGraphqlClient = &customGraphqlClient{
		responses: testCase.graphQLResponses,
	}

	generated.ResetConfig()

	defer func() {
		//Delete temp folder
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}

		rec := recover()
		if rec != nil {
			t.Fatalf("Unexpected panic in testCase %s. Message: %s. Stack: %s", testCase.name, rec, string(debug.Stack()))
		}
	}()

	err = (&spaceCmd{
		SpaceID:  testCase.spaceID,
		Provider: testCase.provider,
		All:      testCase.all,
	}).RunRemoveCloudDevSpace(nil, testCase.args)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}*/
