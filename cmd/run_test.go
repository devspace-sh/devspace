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
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"k8s.io/client-go/tools/clientcmd"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type runTestCase struct {
	name string

	fakeConfig       *latest.Config
	fakeKubeConfig   clientcmd.ClientConfig
	files            map[string]interface{}
	graphQLResponses []interface{}
	providerList     []*cloudlatest.Provider
	answers          []string
	args             []string

	globalFlags flags.GlobalFlags

	expectedOutput string
	expectedErr    string
}

func TestRun(t *testing.T) {
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

	_, err = os.Open("doesn'tExist")
	noFileFoundError := strings.TrimPrefix(err.Error(), "open doesn'tExist: ")

	fsutil.WriteToFile([]byte(""), "someFakeDir")
	err = fsutil.WriteToFile([]byte(""), "someFakeDir/someFile")
	parentDirIsFileErr := strings.TrimPrefix(err.Error(), "mkdir someFakeDir: ")

	testCases := []runTestCase{
		runTestCase{
			name:        "No devspace.yaml",
			expectedErr: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		runTestCase{
			name: "Only configs.yaml exists",
			files: map[string]interface{}{
				constants.DefaultConfigsPath: "",
			},
			expectedErr: "open devspace.yaml: " + noFileFoundError,
		},
		runTestCase{
			name: "Unparsable devspace.yaml",
			files: map[string]interface{}{
				constants.DefaultConfigPath: "unparsable",
			},
			expectedErr: "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into map[interface {}]interface {}",
		},
		runTestCase{
			name: "Unparsable generated.yaml",
			files: map[string]interface{}{
				constants.DefaultConfigPath: map[interface{}]interface{}{},
				generated.ConfigPath:        "unparsable",
			},
			expectedErr: "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into generated.Config",
		},
		runTestCase{
			name: "Invalid version",
			files: map[string]interface{}{
				constants.DefaultConfigPath: &latest.Config{
					Version: "invalid",
				},
			},
			expectedErr: "Unrecognized config version invalid. Please upgrade devspace with `devspace upgrade`",
		},
		runTestCase{
			name: "generated.yaml can't be saved",
			files: map[string]interface{}{
				constants.DefaultConfigPath: &latest.Config{
					Version: latest.Version,
				},
				".devspace": "",
			},
			expectedErr: fmt.Sprintf("mkdir %s: %s", filepath.Join(dir, ".devspace"), parentDirIsFileErr),
		},
		runTestCase{
			name: "empty command",
			files: map[string]interface{}{
				constants.DefaultConfigPath: &latest.Config{
					Version: latest.Version,
				},
			},
			args:        []string{""},
			expectedErr: "execute command: Couldn't find command '' in devspace.yaml",
		},
		runTestCase{
			name: "shell error",
			files: map[string]interface{}{
				constants.DefaultConfigPath: &latest.Config{
					Version: latest.Version,
					Commands: []*latest.CommandConfig{
						&latest.CommandConfig{
							Name:    "exit",
							Command: "exit",
						},
					},
				},
			},
			args:        []string{"exit", "1"},
			expectedErr: "exit code 1",
		},
		runTestCase{
			name: "command error",
			files: map[string]interface{}{
				constants.DefaultConfigPath: &latest.Config{
					Version: latest.Version,
					Commands: []*latest.CommandConfig{
						&latest.CommandConfig{
							Name:    "fail",
							Command: "ThisCommandDoesntExist",
						},
					},
				},
			},
			args:        []string{"fail"},
			expectedErr: "exit code 127",
		},
		runTestCase{
			name: "successful run",
			files: map[string]interface{}{
				constants.DefaultConfigPath: &latest.Config{
					Version: latest.Version,
					Commands: []*latest.CommandConfig{
						&latest.CommandConfig{
							Name:    "exit",
							Command: "exit",
						},
					},
				},
			},
			args: []string{"exit", "0"},
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testRun(t, testCase)
	}
}

func testRun(t *testing.T, testCase runTestCase) {
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

	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	providerConfig, err := cloudconfig.ParseProviderConfig()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	configutil.SetFakeConfig(testCase.fakeConfig)
	configutil.ResetConfig()
	generated.ResetConfig()
	kubeconfig.SetFakeConfig(testCase.fakeKubeConfig)

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	err = (&RunCmd{
		GlobalFlags: &testCase.globalFlags,
	}).RunRun(nil, testCase.args)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
