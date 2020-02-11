package list

/*import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	homedir "github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"

	"gotest.tools/assert"
)

type listProvidersTestCase struct {
	name string

	graphQLResponses    []interface{}
	providerYamlContent interface{}

	expectTablePrint bool
	expectedHeader   []string
	expectedValues   [][]string
	expectedErr      string
}

func TestListProviders(t *testing.T) {
	claimAsJSON, _ := json.Marshal(token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
	})
	validEncodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(validEncodedClaim), "=") {
		validEncodedClaim = strings.TrimSuffix(validEncodedClaim, "=")
	}

	testCases := []listProvidersTestCase{
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
			expectedHeader: []string{"Name", "IsDefault", "Host", "Is logged in"},
			expectedValues: [][]string{
				[]string{"someProvider", "false", "someHost", "false"},
				[]string{"app.devspace.cloud", "false", "https://app.devspace.cloud", "false"},
			},
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

	log.SetInstance(log.Discard)

	for _, testCase := range testCases {
		testListProviders(t, testCase)
	}
}

func testListProviders(t *testing.T, testCase listProvidersTestCase) {
	log.SetFakePrintTable(func(s log.Logger, header []string, values [][]string) {
		assert.Assert(t, testCase.expectTablePrint || len(testCase.expectedHeader)+len(testCase.expectedValues) > 0, "PrintTable unexpectedly called in testCase %s", testCase.name)
		assert.Equal(t, reflect.DeepEqual(header, testCase.expectedHeader), true, "Unexpected header in testCase %s. Expected:%v\nActual:%v", testCase.name, testCase.expectedHeader, header)
		assert.Equal(t, reflect.DeepEqual(values, testCase.expectedValues), true, "Unexpected values in testCase %s. Expected:%v\nActual:%v", testCase.name, testCase.expectedValues, values)
	})

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

	err := (&providersCmd{}).RunListProviders(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}*/
