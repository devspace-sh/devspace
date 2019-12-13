package remove

/*
import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type removeDeploymentTestCase struct {
	name string

	fakeConfig *latest.Config

	args      []string
	answers   []string
	removeAll bool
	files     map[string]interface{}

	expectedErr      string
	expectConfigFile bool
}

func TestRunRemoveDeployment(t *testing.T) {
	testCases := []removeDeploymentTestCase{
		removeDeploymentTestCase{
			name:             "Remove all zero deployments",
			fakeConfig:       &latest.Config{},
			removeAll:        true,
			answers:          []string{"no"},
			expectConfigFile: true,
		},
		removeDeploymentTestCase{
			name: "Remove existent deployment",
			fakeConfig: &latest.Config{
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: "Exists",
					},
				},
			},
			args:             []string{"Exists"},
			answers:          []string{"no"},
			expectConfigFile: true,
		},
		removeDeploymentTestCase{
			name: "Remove all one deployments",
			fakeConfig: &latest.Config{
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: "Exists",
					},
				},
			},
			removeAll:        true,
			answers:          []string{"no"},
			expectConfigFile: true,
		},
	}

	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testRunRemoveDeployment(t, testCase)
	}
}

func testRunRemoveDeployment(t *testing.T, testCase removeDeploymentTestCase) {
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

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	if testCase.fakeConfig == nil {
		loader.ResetConfig()
	} else {
		loader.SetFakeConfig(testCase.fakeConfig)
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

	err = (&deploymentCmd{
		RemoveAll:   testCase.removeAll,
		GlobalFlags: &flags.GlobalFlags{},
	}).RunRemoveDeployment(nil, testCase.args)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}

	err = os.Remove(constants.DefaultConfigPath)
	assert.Equal(t, !os.IsNotExist(err), testCase.expectConfigFile, "Unexpectedly saved or not saved in testCase %s", testCase.name)

}
*/
