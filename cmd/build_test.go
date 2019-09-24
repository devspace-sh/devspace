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
	"github.com/devspace-cloud/devspace/pkg/util/command"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
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
	expectedErr    string
}

func TestBuild(t *testing.T) {
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

	err = command.NewStreamCommand(" ", []string{}).Run(nil, nil, nil)
	pathVarKey := strings.TrimPrefix(err.Error(), "exec: \" \": executable file not found in ")

	testCases := []buildTestCase{
		buildTestCase{
			name:        "config doesn't exist",
			expectedErr: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		buildTestCase{
			name:       "Unparsable generated.yaml",
			fakeConfig: &latest.Config{},
			files: map[string]interface{}{
				".devspace/generated.yaml": "unparsable",
			},
			expectedErr: "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into generated.Config",
		},
		buildTestCase{
			name: "Unparsable devspace.yaml",
			files: map[string]interface{}{
				"devspace.yaml": "unparsable",
			},
			expectedErr: "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into map[interface {}]interface {}",
		},
		buildTestCase{
			name: "Circle dependency",
			fakeConfig: &latest.Config{
				Version: "v1beta3",
				Dependencies: []*latest.DependencyConfig{
					&latest.DependencyConfig{
						Source: &latest.SourceConfig{
							Path: "dependency1",
						},
					},
				},
			},
			files: map[string]interface{}{
				"devspace.yaml": &latest.Config{
					Version: "v1beta3",
					Dependencies: []*latest.DependencyConfig{
						&latest.DependencyConfig{
							Source: &latest.SourceConfig{
								Path: "dependency1",
							},
						},
					},
				},
				"dependency1/devspace.yaml": &latest.Config{
					Version: "v1beta3",
					Dependencies: []*latest.DependencyConfig{
						&latest.DependencyConfig{
							Source: &latest.SourceConfig{
								Path: "..",
							},
						},
					},
				},
			},
			expectedErr:    fmt.Sprintf("build dependencies: Cyclic dependency found: \n%s\n%s\n%s.\n To allow cyclic dependencies run with the '%s' flag", filepath.Join(dir, "dependency1"), dir, filepath.Join(dir, "dependency1"), ansi.Color("--allow-cyclic", "white+b")),
			expectedOutput: "\nInfo Start resolving dependencies",
		},
		buildTestCase{
			name: "1 undeployable image",
			fakeConfig: &latest.Config{
				Images: map[string]*latest.ImageConfig{
					"buildThis": &latest.ImageConfig{
						Image: "someImage",
						Tag:   "someTag",
						Build: &latest.BuildConfig{
							Custom: &latest.CustomConfig{
								Command: " ",
							},
						},
					},
				},
			},
			buildSequentialFlag: true,
			expectedErr:         fmt.Sprintf("build images: Error building image: exec: \" \": executable file not found in %s", pathVarKey),
			expectedOutput:      "\nInfo Build someImage:someTag with custom command   someImage:someTag",
		},
		buildTestCase{
			name: "Deploy 1 image that is too big (Or manipulate the error message to pretend to)",
			fakeConfig: &latest.Config{
				Version: "v1beta3",
				Images: map[string]*latest.ImageConfig{
					"buildThis": &latest.ImageConfig{
						Image: "someImage",
						Tag:   "someTag",
						Build: &latest.BuildConfig{
							Custom: &latest.CustomConfig{
								Command: " no space left on device ", //It's a bit dirty. Force specific kind of error
							},
						},
					},
				},
			},
			buildSequentialFlag: true,
			expectedErr:         fmt.Sprintf("Error building image: Error building image: exec: \" no space left on device \": executable file not found in %s\n\n Try running `%s` to free docker daemon space and retry", pathVarKey, ansi.Color("devspace cleanup images", "white+b")),
			expectedOutput:      "\nInfo Build someImage:someTag with custom command  no space left on device  someImage:someTag",
		},
		buildTestCase{
			name:           "Nothing to build",
			fakeConfig:     &latest.Config{},
			expectedOutput: "\nInfo No images to rebuild. Run with -b to force rebuilding",
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

	generated.ResetConfig()
	if testCase.fakeConfig == nil {
		configutil.ResetConfig()
	} else {
		configutil.SetFakeConfig(testCase.fakeConfig)
	}

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	err = (&BuildCmd{
		GlobalFlags:             &flags.GlobalFlags{},
		SkipPush:                testCase.skipPushFlag,
		AllowCyclicDependencies: testCase.allowCyclicDependenciesFlag,

		ForceBuild:        testCase.forceBuildFlag,
		BuildSequential:   testCase.buildSequentialFlag,
		ForceDependencies: testCase.forceBuildFlag,
	}).Run(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
