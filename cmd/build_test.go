package cmd

/*import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"k8s.io/client-go/tools/clientcmd"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type buildTestCase struct {
	name string

	fakeConfig           *latest.Config
	fakeKubeConfig       clientcmd.ClientConfig
	files                map[string]interface{}
	generatedYamlContent interface{}
	graphQLResponses     []interface{}
	providerList         []*cloudlatest.Provider

	skipPushFlag                bool
	allowCyclicDependenciesFlag bool
	forceBuildFlag              bool
	buildSequentialFlag         bool
	forceDependenciesFlag       bool

	expectedErr string
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

	//err = command.NewStreamCommand(" ", []string{}).Run(nil, nil, nil)
	//pathVarKey := strings.TrimPrefix(err.Error(), "exec: \" \": executable file not found in ")

	testCases := []buildTestCase{
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
			fakeKubeConfig: &customKubeConfig{
				rawconfig: clientcmdapi.Config{},
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
		},
		buildTestCase{
			name: "1 undeployable image",
			files: map[string]interface{}{
				constants.DefaultConfigPath: latest.Config{
					Version: latest.Version,
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
			},
			buildSequentialFlag: true,
			expectedErr:         fmt.Sprintf("build images: Error building image: exec: \" \": executable file not found in %s", pathVarKey),
		},
		buildTestCase{
			name: "Deploy 1 image that is too big (Or manipulate the error message to pretend to)",
			files: map[string]interface{}{
				constants.DefaultConfigPath: latest.Config{
					Version: latest.Version,
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
			},
			buildSequentialFlag: true,
			expectedErr:         fmt.Sprintf("Error building image: Error building image: exec: \" no space left on device \": executable file not found in %s\n\n Try running `%s` to free docker daemon space and retry", pathVarKey, ansi.Color("devspace cleanup images", "white+b")),
		},
		buildTestCase{
			name: "Nothing to build",
			files: map[string]interface{}{
				constants.DefaultConfigPath: latest.Config{
					Version: latest.Version,
				},
			},
		},
	}

	log.OverrideRuntimeErrorHandler(true)
	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testBuild(t, testCase)
	}
}

func testBuild(t *testing.T, testCase buildTestCase) {
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

	providerConfig, err := cloudconfig.Load()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	generated.ResetConfig()
	loader.SetFakeConfig(testCase.fakeConfig)
	loader.ResetConfig()
	kubeconfig.SetFakeConfig(testCase.fakeKubeConfig)

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

	err = filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
		os.RemoveAll(path)
		return nil
	})
	assert.NilError(t, err, "Error cleaning up in testCase %s", testCase.name)
}
*/
