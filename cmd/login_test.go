package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
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

	expectedOutput string
	expectedPanic  string
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
			name:       "Unparsable providerConfig",
			fakeConfig: &latest.Config{},
			files: map[string]interface{}{
				"providerConfig": "unparsable",
			},
			expectedOutput: "",
			expectedPanic:  "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into latest.Config",
		},
		loginTestCase{
			name:       "No key, default provider doesn't exist",
			fakeConfig: &latest.Config{},
			files: map[string]interface{}{
				"providerConfig": &cloudlatest.Config{
					Default: "doesn'tExist",
				},
			},
			expectedOutput: "",
			expectedPanic:  "Error logging in: Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: app.devspace.cloud ",
		},
		loginTestCase{
			name:       "Can't login with key",
			fakeConfig: &latest.Config{},
			files: map[string]interface{}{
				"providerConfig": &cloudlatest.Config{},
			},
			graphQLResponses: []interface{}{
				fmt.Errorf("Custom server error"),
			},
			keyFlag:        "someKey",
			providerFlag:   "app.devspace.cloud",
			expectedOutput: "",
			expectedPanic:  "Error logging in: Access denied for key someKey: Custom server error",
		},
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
			expectedOutput: "\nDone Successfully logged into app.devspace.cloud\nWarn Error logging into docker registries: get registries: Custom server error\nInfo Successful logged into app.devspace.cloud",
		},
	}

	homedir, err := homedir.Dir()
	assert.NilError(t, err, "Error getting homedir")
	relDir, err := filepath.Rel(homedir, dir)
	assert.NilError(t, err, "Error getting relative dir path")
	cloudconfig.DevSpaceProvidersConfigPath = filepath.Join(relDir, "providerConfig")
	cloudconfig.LegacyDevSpaceCloudConfigPath = filepath.Join(relDir, "providerCloudConfig")

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testLogin(t, testCase)
	}
}

func testLogin(t *testing.T, testCase loginTestCase) {
	logOutput = ""

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

	configutil.SetFakeConfig(testCase.fakeConfig)
	generated.ResetConfig()
	cloudconfig.Reset()

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	(&LoginCmd{
		Key:      testCase.keyFlag,
		Provider: testCase.providerFlag,
	}).RunLogin(nil, []string{})
}
