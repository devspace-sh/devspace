package list

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"gopkg.in/yaml.v2"

	"gotest.tools/assert"
)

type listProfilesTestCase struct {
	name string

	fakeConfig       *latest.Config
	graphQLResponses []interface{}
	files            map[string]interface{}

	expectedOutput string
	expectedErr    string
}

func TestListProfiles(t *testing.T) {
	claimAsJSON, _ := json.Marshal(token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
	})
	validEncodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(validEncodedClaim), "=") {
		validEncodedClaim = strings.TrimSuffix(validEncodedClaim, "=")
	}

	expectedHeader := ansi.Color(" Name  ", "green+b") + "       " + ansi.Color(" Active  ", "green+b")
	testCases := []listProfilesTestCase{
		listProfilesTestCase{
			name:       "print 1 profile",
			fakeConfig: &latest.Config{},
			files: map[string]interface{}{
				"devspace.yaml": map[interface{}]interface{}{
					"profiles": []interface{}{
						map[interface{}]interface{}{
							"name": "someProfile",
						},
					},
				},
			},
			expectedOutput: "\n" + expectedHeader + "\n someProfile   false   \n\n",
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testListProfiles(t, testCase)
	}
}

func testListProfiles(t *testing.T, testCase listProfilesTestCase) {
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

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	configutil.SetFakeConfig(testCase.fakeConfig)
	generated.ResetConfig()

	err = (&profilesCmd{}).RunListProfiles(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
