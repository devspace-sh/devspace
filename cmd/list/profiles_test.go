package list

/*
import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"gopkg.in/yaml.v2"

	"gotest.tools/assert"
)

type listProfilesTestCase struct {
	name string

	fakeConfig       *latest.Config
	graphQLResponses []interface{}
	files            map[string]interface{}

	expectTablePrint bool
	expectedHeader   []string
	expectedValues   [][]string
	expectedErr      string
}

func TestListProfiles(t *testing.T) {
	claimAsJSON, _ := json.Marshal(token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
	})
	validEncodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(validEncodedClaim), "=") {
		validEncodedClaim = strings.TrimSuffix(validEncodedClaim, "=")
	}

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
			expectedHeader: []string{"Name", "Active"},
			expectedValues: [][]string{
				[]string{"someProfile", "false"},
			},
		},
	}

	log.SetInstance(log.Discard)

	for _, testCase := range testCases {
		testListProfiles(t, testCase)
	}
}

func testListProfiles(t *testing.T, testCase listProfilesTestCase) {
	log.SetFakePrintTable(func(s log.Logger, header []string, values [][]string) {
		assert.Assert(t, testCase.expectTablePrint || len(testCase.expectedHeader)+len(testCase.expectedValues) > 0, "PrintTable unexpectedly called in testCase %s", testCase.name)
		assert.Equal(t, reflect.DeepEqual(header, testCase.expectedHeader), true, "Unexpected header in testCase %s. Expected:%v\nActual:%v", testCase.name, testCase.expectedHeader, header)
		assert.Equal(t, reflect.DeepEqual(values, testCase.expectedValues), true, "Unexpected values in testCase %s. Expected:%v\nActual:%v", testCase.name, testCase.expectedValues, values)
	})

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

	loader.SetFakeConfig(testCase.fakeConfig)
	generated.ResetConfig()

	err = (&profilesCmd{}).RunListProfiles(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}*/
