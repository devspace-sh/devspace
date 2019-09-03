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
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/mgutz/ansi"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type buildTestCase struct {
	name string

	fakeConfig           *latest.Config
	files                map[string]interface{}
	generatedYamlContent interface{}
	graphQLResponses     []interface{}
	providerList         []*cloudlatest.Provider

	skipPushFlag                bool
	allowCyclicDependenciesFlag bool
	forceBuildFlag              bool
	buildSequentialFlag         bool
	forceDependenciesFlag       bool

	expectedOutput string
	expectedPanic  string
}

func TestBuild(t *testing.T) {
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

	_, err = os.Open("doesn'tExist")
	noFileFoundError := strings.TrimPrefix(err.Error(), "open doesn'tExist: ")

	testCases := []buildTestCase{
		buildTestCase{
			name:          "config doesn't exist",
			expectedPanic: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		buildTestCase{
			name:       "Unparsable generated.yaml",
			fakeConfig: &latest.Config{},
			files: map[string]interface{}{
				".devspace/generated.yaml": "unparsable",
			},
			expectedPanic: "Error loading generated.yaml: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into generated.Config",
		},
		buildTestCase{
			name:          "No devspace.yaml",
			fakeConfig:    &latest.Config{},
			expectedPanic: fmt.Sprintf("Loading config: open devspace.yaml: %s", noFileFoundError),
		},
		buildTestCase{
			name: "generated.yaml can't be saved",
			files: map[string]interface{}{
				"devspace.yaml":                     &latest.Config{},
				".devspace/generated.yaml/someFile": "",
			},
			expectedPanic:  fmt.Sprintf("Couldn't save generated config: open %s: is a directory", filepath.Join(dir, ".devspace/generated.yaml")),
			expectedOutput: "\nInfo Loaded config from devspace.yaml",
		},
		buildTestCase{
			name: "Circle dependency",
			files: map[string]interface{}{
				"devspace.yaml": &latest.Config{
					Dependencies: &[]*latest.DependencyConfig{
						&latest.DependencyConfig{
							Source: &latest.SourceConfig{
								Path: ptr.String("dependency1"),
							},
						},
					},
				},
				"dependency1/devspace.yaml": &latest.Config{
					Dependencies: &[]*latest.DependencyConfig{
						&latest.DependencyConfig{
							Source: &latest.SourceConfig{
								Path: ptr.String(".."),
							},
						},
					},
				},
			},
			expectedPanic:  fmt.Sprintf("Error deploying dependencies: Cyclic dependency found: \n%s\n%s\n%s.\n To allow cyclic dependencies run with the '%s' flag", filepath.Join(dir, "dependency1"), dir, filepath.Join(dir, "dependency1"), ansi.Color("--allow-cyclic", "white+b")),
			expectedOutput: "\nInfo Loaded config from devspace.yaml\nWait Resolving dependencies",
		},
		buildTestCase{
			name: "1 undeployable image",
			files: map[string]interface{}{
				"devspace.yaml": &latest.Config{
					Images: &map[string]*latest.ImageConfig{
						"buildThis": &latest.ImageConfig{
							Image: ptr.String("someImage"),
						},
					},
				},
			},
			expectedPanic:  fmt.Sprintf("Error building image: Error during shouldRebuild check: Dockerfile ./Dockerfile missing: CreateFile ./Dockerfile: %s", noFileFoundError),
			expectedOutput: "\nInfo Loaded config from devspace.yaml",
		},
		buildTestCase{
			name: "Deploy 1 image that is too big (Or manipulate the error message to pretend to)",
			files: map[string]interface{}{
				"devspace.yaml": &latest.Config{
					Images: &map[string]*latest.ImageConfig{
						"buildThis": &latest.ImageConfig{
							Image:      ptr.String("someImage"),
							Dockerfile: ptr.String("no space left on device"), //It's a bit dirty. Force specific kind of error
						},
					},
				},
			},
			expectedPanic:  fmt.Sprintf("Error building image: Error during shouldRebuild check: Dockerfile no space left on device missing: CreateFile no space left on device: The system cannot find the file specified.\n\n Try running `%s` to free docker daemon space and retry", ansi.Color("devspace cleanup images", "white+b")),
			expectedOutput: "\nInfo Loaded config from devspace.yaml",
		},
		buildTestCase{
			name:       "Nothing to build",
			fakeConfig: &latest.Config{},
			files: map[string]interface{}{
				"devspace.yaml": &latest.Config{},
			},
			expectedOutput: "\nInfo Loaded config from devspace.yaml\nInfo No images to rebuild. Run with -b to force rebuilding",
		},
	}

	//The build-command wants to overwrite error logging with file logging. This workaround prevents that.
	err = os.MkdirAll(log.Logdir+"errors.log", 0700)
	assert.NilError(t, err, "Error overwriting log file before its creation")
	log.OverrideRuntimeErrorHandler()

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testBuild(t, testCase)
	}
}

func testBuild(t *testing.T, testCase buildTestCase) {
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

	(&BuildCmd{
		SkipPush:                testCase.skipPushFlag,
		AllowCyclicDependencies: testCase.allowCyclicDependenciesFlag,

		ForceBuild:        testCase.forceBuildFlag,
		BuildSequential:   testCase.buildSequentialFlag,
		ForceDependencies: testCase.forceBuildFlag,
	}).Run(nil, []string{})
}
