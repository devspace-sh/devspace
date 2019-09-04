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

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type deployTestCase struct {
	name string

	fakeConfig           *latest.Config
	files                map[string]interface{}
	generatedYamlContent interface{}
	graphQLResponses     []interface{}
	providerList         []*cloudlatest.Provider

	namespaceFlag               string
	kubeContextFlag             string
	dockerTargetFlag            string
	forceBuildFlag              bool
	skipBuildFlag               bool
	buildSequentialFlag         bool
	forceDeployFlag             bool
	deploymentsFlag             string
	forceDependenciesFlag       bool
	switchContextFlag           bool
	skipPushFlag                bool
	allowCyclicDependenciesFlag bool

	expectedOutput string
	expectedPanic  string
}

func TestDeploy(t *testing.T) {
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

	_, err = os.Open("doesn'tExist")
	noFileFoundError := strings.TrimPrefix(err.Error(), "open doesn'tExist: ")

	testCases := []deployTestCase{
		deployTestCase{
			name:          "config doesn't exist",
			expectedPanic: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		deployTestCase{
			name:           "Invalid flags",
			fakeConfig:     &latest.Config{},
			skipBuildFlag:  true,
			forceBuildFlag: true,
			expectedPanic:  "Flags --skip-build & --force-build cannot be used together",
		},
		deployTestCase{
			name:       "Unparsable generated.yaml",
			fakeConfig: &latest.Config{},
			files: map[string]interface{}{
				".devspace/generated.yaml": "unparsable",
			},
			expectedPanic: "Error loading generated.yaml: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into generated.Config",
		},
		deployTestCase{
			name:          "No devspace.yaml",
			fakeConfig:    &latest.Config{},
			expectedPanic: fmt.Sprintf("Loading config: open devspace.yaml: %s", noFileFoundError),
		},
		deployTestCase{
			name: "generated.yaml is a dir",
			files: map[string]interface{}{
				"devspace.yaml":                     &latest.Config{},
				".devspace/generated.yaml/someFile": "",
			},
			namespaceFlag:   "someNamespace",
			kubeContextFlag: "someKubeContext",
			expectedPanic:   fmt.Sprintf("Couldn't save generated config: open %s: is a directory", filepath.Join(dir, ".devspace/generated.yaml")),
			expectedOutput:  "\nInfo Loaded config from devspace.yaml\nInfo Using someNamespace namespace for deploying\nInfo Using someKubeContext kube context for deploying",
		},
		deployTestCase{
			name: "cloud space can't be resumed",
			files: map[string]interface{}{
				"devspace.yaml": &latest.Config{},
				".devspace/generated.yaml": &generated.Config{
					CloudSpace: &generated.CloudSpaceConfig{
						KubeContext: "someKubeContext",
					},
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
			expectedPanic:  "Error retrieving Spaces details: Custom graphQL error",
			expectedOutput: "\nInfo Loaded config from devspace.yaml",
		},
	}

	//The deploy-command wants to overwrite error logging with file logging. This workaround prevents that.
	err = os.MkdirAll(log.Logdir+"errors.log", 0700)
	assert.NilError(t, err, "Error overwriting log file before its creation")
	log.OverrideRuntimeErrorHandler()

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testDeploy(t, testCase)
	}
}

func testDeploy(t *testing.T, testCase deployTestCase) {
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

	providerConfig, err := cloudconfig.ParseProviderConfig()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	configutil.SetFakeConfig(testCase.fakeConfig)
	generated.ResetConfig()

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	(&DeployCmd{
		Namespace:    testCase.namespaceFlag,
		KubeContext:  testCase.kubeContextFlag,
		DockerTarget: testCase.dockerTargetFlag,

		ForceBuild:        testCase.forceBuildFlag,
		SkipBuild:         testCase.skipBuildFlag,
		BuildSequential:   testCase.buildSequentialFlag,
		ForceDeploy:       testCase.forceDeployFlag,
		Deployments:       testCase.deploymentsFlag,
		ForceDependencies: testCase.forceDependenciesFlag,

		SwitchContext: testCase.switchContextFlag,
		SkipPush:      testCase.skipPushFlag,

		AllowCyclicDependencies: testCase.allowCyclicDependenciesFlag,
	}).Run(nil, []string{})
}
