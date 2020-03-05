package cmd

/*import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	homedir "github.com/mitchellh/go-homedir"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type loginTestCase struct {
	name string

	fakeConfig           *latest.Config
	files                map[string]interface{}
	generatedYamlContent interface{}
	graphQLResponses     []interface{}

	keyFlag      string
	providerFlag string

	expectedErr    string
}

func TestLogin(t *testing.T) {
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
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
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

	testCases := []loginTestCase{
		loginTestCase{
			name:       "Successful relogin with key",
			fakeConfig: &latest.Config{},
			files: map[string]interface{}{
				"providerConfig": &cloudlatest.Config{},
			},
			graphQLResponses: []interface{}{
				struct {
					Spaces []*interface{} `json:"space"`
				}{
					Spaces: []*interface{}{},
				},
				fmt.Errorf("Custom server error"),
			},
			keyFlag:        "someKey",
			providerFlag:   "app.devspace.cloud",
		},
	}

	homedir, err := homedir.Dir()
	assert.NilError(t, err, "Error getting homedir")
	relDir, err := filepath.Rel(homedir, dir)
	assert.NilError(t, err, "Error getting relative dir path")
	cloudconfig.DevSpaceProvidersConfigPath = filepath.Join(relDir, "providerConfig")
	cloudconfig.LegacyDevSpaceCloudConfigPath = filepath.Join(relDir, "providerCloudConfig")

	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testLogin(t, testCase)
	}
}

func testLogin(t *testing.T, testCase loginTestCase) {
	defer func() {
		for path := range testCase.files {
			removeTask := strings.Split(path, "/")[0]
			err := os.RemoveAll(removeTask)
			assert.NilError(t, err, "Error cleaning up folder in testCase %s", testCase.name)
		}
		err := os.RemoveAll(log.Logdir)
		assert.NilError(t, err, "Error cleaning up folder in testCase %s", testCase.name)
	}()

	cloudpkg.DefaultGraphqlClient = &customGraphqlClient{
		responses: testCase.graphQLResponses,
	}

	loader.SetFakeConfig(testCase.fakeConfig)
	generated.ResetConfig()
	cloudconfig.Reset()

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	err := (&LoginCmd{
		Key:      testCase.keyFlag,
		Provider: testCase.providerFlag,
	}).RunLogin(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}*/
