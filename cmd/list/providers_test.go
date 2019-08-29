package list

import (
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
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	homedir "github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"

	"gotest.tools/assert"
)

type listProvidersTestCase struct {
	name string

	graphQLResponses    []interface{}
	providerYamlContent interface{}

	expectedOutput string
	expectedPanic  string
}

func TestListProviders(t *testing.T) {
	claimAsJSON, _ := json.Marshal(token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
	})
	validEncodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(validEncodedClaim), "=") {
		validEncodedClaim = strings.TrimSuffix(validEncodedClaim, "=")
	}

	expectedHeader := ansi.Color(" Name  ", "green+b") + "              " + ansi.Color(" IsDefault  ", "green+b") + ansi.Color(" Host  ", "green+b") + "                      " + ansi.Color(" Is logged in  ", "green+b")
	testCases := []listProvidersTestCase{
		listProvidersTestCase{
			name:                "Provider can't be parsed",
			providerYamlContent: "unparsable",
			expectedPanic:       "Error loading provider config: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into latest.Config",
		},
		listProvidersTestCase{
			name: "One provider",
			providerYamlContent: &cloudlatest.Config{
				Providers: []*cloudlatest.Provider{
					&cloudlatest.Provider{
						Name: "someProvider",
						Host: "someHost",
					},
				},
			},
			expectedOutput: "\n" + expectedHeader + "\n someProvider         false       someHost                     false         \n app.devspace.cloud   false       https://app.devspace.cloud   false         \n\n",
		},
	}

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
	assert.NilError(t, err, "Error getting homedir")
	relDir, err := filepath.Rel(homedir, dir)
	assert.NilError(t, err, "Error getting relative dir path")
	cloudconfig.DevSpaceProvidersConfigPath = filepath.Join(relDir, "providerConfig")
	cloudconfig.LegacyDevSpaceCloudConfigPath = filepath.Join(relDir, "providerCloudConfig")

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
	}()

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testListProviders(t, testCase)
	}
}

func testListProviders(t *testing.T, testCase listProvidersTestCase) {
	logOutput = ""

	cloudpkg.DefaultGraphqlClient = &customGraphqlClient{
		responses: testCase.graphQLResponses,
	}

	if testCase.providerYamlContent != nil {
		content, err := yaml.Marshal(testCase.providerYamlContent)
		assert.NilError(t, err, "Error parsing providers.yaml to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(content, "providerConfig")
		assert.NilError(t, err, "Error writing provider.yaml in testCase %s", testCase.name)
	}

	cloudconfig.Reset()

	defer func() {
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

	(&providersCmd{}).RunListProviders(nil, []string{})

	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
