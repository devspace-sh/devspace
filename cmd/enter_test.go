package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"k8s.io/client-go/tools/clientcmd"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type enterTestCase struct {
	name string

	fakeConfig           *latest.Config
	fakeKubeConfig       clientcmd.ClientConfig
	files                map[string]interface{}
	generatedYamlContent interface{}
	graphQLResponses     []interface{}
	providerList         []*cloudlatest.Provider

	containerFlag     string
	labelSelectorFlag string
	namespaceFlag     string
	podFlag           string
	pickFlag          bool

	expectedOutput string
	expectedErr    string
}

func TestEnter(t *testing.T) {
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

	testCases := []enterTestCase{
		enterTestCase{
			name:       "Invalid kube config",
			fakeConfig: &latest.Config{},
			fakeKubeConfig: &customKubeConfig{
				rawConfigError: fmt.Errorf("RawConfigError"),
			},
			expectedErr: "new kube client: RawConfigError",
		},
		/*enterTestCase{
			name:       "Unparsable generated.yaml",
			fakeConfig: &latest.Config{},
			files: map[string]interface{}{
				".devspace/generated.yaml": "unparsable",
			},
			expectedErr: "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into generated.Config",
		},
		enterTestCase{
			name: "cloud space can't be resumed",
			files: map[string]interface{}{
				"devspace.yaml":            &latest.Config{},
				".devspace/generated.yaml": &generated.Config{
				},
			},
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Key: "someKey",
				},
			},
			graphQLResponses: []interface{}{
				fmt.Errorf("Custom graphQL error"),
			},
			expectedErr: "Error retrieving Spaces details: Custom graphQL error",
		},*/
	}

	//The dev-command wants to overwrite error logging with file logging. This workaround prevents that.
	err = os.MkdirAll(log.Logdir+"errors.log", 0700)
	assert.NilError(t, err, "Error overwriting log file before its creation")
	log.OverrideRuntimeErrorHandler()

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testEnter(t, testCase)
	}
}

func testEnter(t *testing.T, testCase enterTestCase) {
	logOutput = ""

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

	providerConfig, err := cloudconfig.ParseProviderConfig()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	configutil.SetFakeConfig(testCase.fakeConfig)
	generated.ResetConfig()
	kubeconfig.SetFakeConfig(testCase.fakeKubeConfig)

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	err = (&EnterCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: testCase.namespaceFlag,
		},
		Container:     testCase.containerFlag,
		LabelSelector: testCase.labelSelectorFlag,
		Pod:           testCase.podFlag,
		Pick:          testCase.pickFlag,
	}).Run(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
