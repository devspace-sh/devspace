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
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"k8s.io/client-go/tools/clientcmd"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type purgeTestCase struct {
	name string

	fakeConfig       *latest.Config
	fakeKubeConfig   clientcmd.ClientConfig
	files            map[string]interface{}
	graphQLResponses []interface{}
	providerList     []*cloudlatest.Provider
	answers          []string

	deploymentsFlag             string
	allowCyclicDependenciesFlag bool
	verboseDependenciesFlag     bool
	purgeDependenciesFlag       bool
	globalFlags                 flags.GlobalFlags

	expectedOutput string
	expectedErr    string
}

func TestPurge(t *testing.T) {
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

	testCases := []purgeTestCase{
		purgeTestCase{
			name:        "No devspace.yaml",
			expectedErr: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		purgeTestCase{
			name:       "Unparsable generated.yaml",
			fakeConfig: &latest.Config{},
			files: map[string]interface{}{
				".devspace/generated.yaml": "unparsable",
			},
			expectedErr: "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into generated.Config",
		},
		purgeTestCase{
			name:       "Invalid global flags",
			fakeConfig: &latest.Config{},
			globalFlags: flags.GlobalFlags{
				KubeContext:   "a",
				SwitchContext: true,
			},
			expectedErr: "Flag --kube-context cannot be used together with --switch-context",
		},
		purgeTestCase{
			name:       "invalid kubeconfig",
			fakeConfig: &latest.Config{},
			fakeKubeConfig: &customKubeConfig{
				rawConfigError: fmt.Errorf("RawConfigError"),
			},
			expectedErr: "create kube client: RawConfigError",
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testPurge(t, testCase)
	}
}

func testPurge(t *testing.T, testCase purgeTestCase) {
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

	err = (&PurgeCmd{
		Deployments:             testCase.deploymentsFlag,
		AllowCyclicDependencies: testCase.allowCyclicDependenciesFlag,
		VerboseDependencies:     testCase.verboseDependenciesFlag,
		PurgeDependencies:       testCase.purgeDependenciesFlag,
		GlobalFlags:             &testCase.globalFlags,
	}).Run(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)

	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
