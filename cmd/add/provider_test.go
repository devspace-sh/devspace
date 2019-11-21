package add

/*import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	homedir "github.com/mitchellh/go-homedir"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type customGraphqlClient struct {
	responses []interface{}
}

func (q *customGraphqlClient) GrapqhlRequest(p *cloudpkg.Provider, request string, vars map[string]interface{}, response interface{}) error {
	if len(q.responses) == 0 {
		panic("Not enough responses. Need response for: " + request)
	}
	currentResponse := q.responses[0]
	q.responses = q.responses[1:]

	errorResponse, isError := currentResponse.(error)
	if isError {
		return errorResponse
	}
	buf, err := json.Marshal(currentResponse)
	if err != nil {
		panic(fmt.Sprintf("Cannot encode response. %d responses left", len(q.responses)))
	}
	json.NewDecoder(bytes.NewReader(buf)).Decode(&response)

	return nil
}

type addProviderTestCase struct {
	name string

	args  []string
	files map[string]interface{}

	expectedErr       string
	expectConfigFile  bool
	expectedProviders []*cloudlatest.Provider
}

func TestRunAddProvider(t *testing.T) {
	testCases := []addProviderTestCase{
		addProviderTestCase{
			name: "Add existing provider",
			files: map[string]interface{}{
				"provider.yaml": cloudlatest.Config{
					Providers: []*cloudlatest.Provider{
						&cloudlatest.Provider{
							Name: "someProvider",
							Key:  "someKey",
						},
					},
				},
			},
			args:        []string{"someProvider"},
			expectedErr: "Provider someProvider does already exist",
			expectedProviders: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name: "someProvider",
					Host: "https://someProvider",
					Key:  "someKey",
				},
			},
		},
	}

	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testRunAddProvider(t, testCase)
	}
}

func testRunAddProvider(t *testing.T, testCase addProviderTestCase) {
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

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	cloudconfig.Reset()
	cloudpkg.DefaultGraphqlClient = &customGraphqlClient{}

	homedir, err := homedir.Dir()
	assert.NilError(t, err, "Error getting homedir testCase %s", testCase.name)
	relDir, err := filepath.Rel(homedir, dir)
	assert.NilError(t, err, "Error getting relative dir path testCase %s", testCase.name)
	cloudconfig.DevSpaceProvidersConfigPath = filepath.Join(relDir, "provider.yaml")
	cloudconfig.LegacyDevSpaceCloudConfigPath = filepath.Join(relDir, "legacy.yaml")

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

	err = (&providerCmd{}).RunAddProvider(nil, testCase.args)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
		return
	}

	config, err := cloudconfig.Load()
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, len(testCase.expectedProviders), len(config.Providers), "Wrong number of providers in testCase %s", testCase.name)
	for index, provider := range config.Providers {
		assert.Equal(t, testCase.expectedProviders[index].Name, provider.Name, "Provider name in index %d unexpected in testCase %s", index, testCase.name)
		assert.Equal(t, testCase.expectedProviders[index].Host, provider.Host, "Provider host in index %d unexpected in testCase %s", index, testCase.name)
		assert.Equal(t, testCase.expectedProviders[index].Key, provider.Key, "Provider key in index %d unexpected in testCase %s", index, testCase.name)
		assert.Equal(t, testCase.expectedProviders[index].Token, provider.Token, "Provider token in index %d unexpected in testCase %s", index, testCase.name)
	}

	err = os.Remove(constants.DefaultConfigPath)
	assert.Equal(t, !os.IsNotExist(err), testCase.expectConfigFile, "Unexpectedly saved or not saved in testCase %s", testCase.name)
}*/
