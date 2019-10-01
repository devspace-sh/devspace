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

type devTestCase struct {
	name string

	fakeConfig           *latest.Config
	fakeKubeConfig       clientcmd.ClientConfig
	files                map[string]interface{}
	generatedYamlContent interface{}
	graphQLResponses     []interface{}
	providerList         []*cloudlatest.Provider

	forceBuildFlag              bool
	skipBuildFlag               bool
	buildSequentialFlag         bool
	forceDeploymentFlag         bool
	deploymentsFlag             string
	forceDependenciesFlag       bool
	skipPushFlag                bool
	allowCyclicDependenciesFlag bool

	syncFlag            bool
	terminalFlag        bool
	exitAfterDeployFlag bool
	skipPipelineFlag    bool
	switchContextFlag   bool
	portForwardingFlag  bool
	verboseSyncFlag     bool
	selectorFlag        string
	containerFlag       string
	labelSelectorFlag   string
	globalFlags         flags.GlobalFlags

	expectedOutput string
	expectedErr    string
}

func TestDev(t *testing.T) {
	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	//_, err = os.Open("doesn'tExist")
	//noFileFoundError := strings.TrimPrefix(err.Error(), "open doesn'tExist: ")

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

	testCases := []devTestCase{
		devTestCase{
			name:        "config doesn't exist",
			expectedErr: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		devTestCase{
			name:           "Invalid flags",
			fakeConfig:     &latest.Config{},
			skipBuildFlag:  true,
			forceBuildFlag: true,
			expectedErr:    "Flags --skip-build & --force-build cannot be used together",
		},
		devTestCase{
			name:       "Invalid global flags",
			fakeConfig: &latest.Config{},
			globalFlags: flags.GlobalFlags{
				KubeContext:   "a",
				SwitchContext: true,
			},
			expectedErr: "Flag --kube-context cannot be used together with --switch-context",
		},
		devTestCase{
			name:       "Unparsable generated.yaml",
			fakeConfig: &latest.Config{},
			files: map[string]interface{}{
				".devspace/generated.yaml": "unparsable",
			},
			expectedErr: "Error loading generated.yaml: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into generated.Config",
		},
		devTestCase{
			name:       "Invalid kube config",
			fakeConfig: &latest.Config{},
			fakeKubeConfig: &customKubeConfig{
				rawConfigError: fmt.Errorf("RawConfigError"),
			},
			expectedErr: "Unable to create new kubectl client: RawConfigError",
		},
		/*devTestCase{
			name:          "No devspace.yaml",
			fakeConfig:    &latest.Config{},
			expectedErr: fmt.Sprintf("Loading config: open devspace.yaml: %s", noFileFoundError),
		},
		devTestCase{
			name: "generated.yaml is a dir",
			files: map[string]interface{}{
				"devspace.yaml":                     &latest.Config{},
				".devspace/generated.yaml/someFile": "",
			},
			namespaceFlag:  "someNamespace",
			expectedErr:  fmt.Sprintf("Couldn't save generated config: open %s: is a directory", filepath.Join(dir, ".devspace/generated.yaml")),
			expectedOutput: "\nInfo Loaded config from devspace.yaml\nInfo Using someNamespace namespace",
		},
		devTestCase{
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
			expectedErr:  "Error retrieving Spaces details: Custom graphQL error",
			expectedOutput: "\nInfo Loaded config from devspace.yaml",
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
		testDev(t, testCase)
	}
}

func testDev(t *testing.T, testCase devTestCase) {
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

	err = (&DevCmd{
		GlobalFlags: &testCase.globalFlags,

		AllowCyclicDependencies: testCase.allowCyclicDependenciesFlag,
		SkipPush:                testCase.skipPushFlag,

		ForceBuild:        testCase.forceBuildFlag,
		SkipBuild:         testCase.skipBuildFlag,
		BuildSequential:   testCase.buildSequentialFlag,
		ForceDeploy:       testCase.forceDeploymentFlag,
		Deployments:       testCase.deploymentsFlag,
		ForceDependencies: testCase.forceDependenciesFlag,

		Sync:            testCase.syncFlag,
		Terminal:        testCase.terminalFlag,
		ExitAfterDeploy: testCase.exitAfterDeployFlag,
		SkipPipeline:    testCase.skipPipelineFlag,
		Portforwarding:  testCase.portForwardingFlag,
		VerboseSync:     testCase.verboseSyncFlag,
	}).Run(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
