package remove

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	homedir "github.com/mitchellh/go-homedir"

	"gotest.tools/assert"
)

type removeContextTestCase struct {
	name string

	fakeConfig *latest.Config

	args             []string
	answers          []string
	graphQLResponses []interface{}
	provider         string
	all              bool
	providerList     []*cloudlatest.Provider

	expectedOutput string
	expectedPanic  string
}

func TestRunRemoveContext(t *testing.T) {
	claimAsJSON, _ := json.Marshal(token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
	})
	validEncodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(validEncodedClaim), "=") {
		validEncodedClaim = strings.TrimSuffix(validEncodedClaim, "=")
	}

	testCases := []removeContextTestCase{
		removeContextTestCase{
			name:          "Cloud context not gettable",
			expectedPanic: "Error getting cloud context: Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: ",
		},
		removeContextTestCase{
			name:     "Spaces not gettable",
			provider: "myProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name: "myProvider",
					Key:  "someKey",
				},
			},
			all: true,
			graphQLResponses: []interface{}{
				fmt.Errorf("TestError from graphql server"),
			},
			expectedPanic: "TestError from graphql server",
		},
		removeContextTestCase{
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
			},
			expectedOutput: "\nDone Deleted kubectl context for space \nDone All space kubectl contexts removed",
		},
		removeContextTestCase{
			name:     "Neither all nor argument specified",
			provider: "myProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name: "myProvider",
					Key:  "someKey",
				},
			},
			expectedPanic: "Please specify a space name or the --all flag",
		},
		removeContextTestCase{
			name:     "Unparsable space name",
			provider: "myProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name: "myProvider",
					Key:  "someKey",
				},
			},
			args:          []string{"a:b:c"},
			expectedPanic: "Error retrieving space a:b:c: Error parsing space name a:b:c: Expected : only once",
		},
		removeContextTestCase{
			name:     "Unparsable space name",
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
			},
			args:           []string{"a:b"},
			expectedOutput: "\nDone Kubectl context deleted for space a:b",
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testRunRemoveContext(t, testCase)
	}
}

func testRunRemoveContext(t *testing.T, testCase removeContextTestCase) {
	logOutput = ""

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

	providerConfig, err := cloudconfig.ParseProviderConfig()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	cloudpkg.DefaultGraphqlClient = &customGraphqlClient{
		responses: testCase.graphQLResponses,
	}

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
		if testCase.expectedPanic == "" {
			if rec != nil {
				t.Fatalf("Unexpected panic in testCase %s. Message: %s. Stack: %s", testCase.name, rec, string(debug.Stack()))
			}
		} else {
			if rec == nil {
				t.Fatalf("Unexpected no panic in testCase %s", testCase.name)
			} else {
				assert.Equal(t, rec, testCase.expectedPanic, "Wrong panic message in testCase %s. Stack: %s", testCase.name, string(debug.Stack()))
			}
		}
		assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
	}()

	(&contextCmd{
		Provider: testCase.provider,
		All:      testCase.all,
	}).RunRemoveContext(nil, testCase.args)

	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
