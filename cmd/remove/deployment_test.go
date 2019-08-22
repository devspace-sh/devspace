package remove

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"gotest.tools/assert"
)

type removeDeploymentTestCase struct {
	name string

	fakeConfig *latest.Config

	args      []string
	answers   []string
	removeAll bool

	expectedOutput   string
	expectedPanic    string
	expectConfigFile bool
}

func TestRunRemoveDeployment(t *testing.T) {
	testCases := []removeDeploymentTestCase{
		removeDeploymentTestCase{
			name:          "No devspace config",
			args:          []string{""},
			expectedPanic: "Couldn't find any devspace configuration. Please run `devspace init`",
		},
		removeDeploymentTestCase{
			name:          "Don't specify what to remove",
			fakeConfig:    &latest.Config{},
			answers:       []string{"no"},
			expectedPanic: "You have to specify either a deployment name or the --all flag",
		},
		/*removeDeploymentTestCase{
			name:          "Wrong kubectl configuration",
			fakeConfig:    &latest.Config{},
			answers:       []string{"yes"},
			expectedPanic: "Unable to create new kubectl client: invalid configuration: no configuration has been provided",
		},*/
		removeDeploymentTestCase{
			name:             "Remove not existent deployment",
			fakeConfig:       &latest.Config{},
			args:             []string{"doesn'tExist"},
			answers:          []string{"no"},
			expectedOutput:   "\nWarn Couldn't find deployment doesn'tExist",
			expectConfigFile: true,
		},
		removeDeploymentTestCase{
			name:             "Remove all zero deployments",
			fakeConfig:       &latest.Config{},
			removeAll:        true,
			answers:          []string{"no"},
			expectedOutput:   "\nWarn Couldn't find any deployment",
			expectConfigFile: true,
		},
		removeDeploymentTestCase{
			name: "Remove existent deployment",
			fakeConfig: &latest.Config{
				Deployments: &[]*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: ptr.String("Exists"),
					},
				},
			},
			args:             []string{"Exists"},
			answers:          []string{"no"},
			expectedOutput:   "\nDone Successfully removed deployment Exists",
			expectConfigFile: true,
		},
		removeDeploymentTestCase{
			name: "Remove all one deployments",
			fakeConfig: &latest.Config{
				Deployments: &[]*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: ptr.String("Exists"),
					},
				},
			},
			removeAll:        true,
			answers:          []string{"no"},
			expectedOutput:   "\nDone Successfully removed all deployments",
			expectConfigFile: true,
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testRunRemoveDeployment(t, testCase)
	}
}

func testRunRemoveDeployment(t *testing.T, testCase removeDeploymentTestCase) {
	logOutput = ""

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

	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	isDeploymentsNil := testCase.fakeConfig == nil || testCase.fakeConfig.Deployments == nil
	configutil.SetFakeConfig(testCase.fakeConfig)
	if isDeploymentsNil && testCase.fakeConfig != nil {
		testCase.fakeConfig.Deployments = nil
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

		rec := recover()
		if testCase.expectedPanic == "" {
			if rec != nil {
				t.Fatalf("Unexpected panic in testCase %s. Message: %s", testCase.name, rec)
			}
		} else {
			if rec == nil {
				t.Fatalf("Unexpected no panic in testCase %s", testCase.name)
			} else {
				assert.Equal(t, rec, testCase.expectedPanic, "Wrong panic message in testCase %s", testCase.name)
			}
		}
		assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
	}()

	(&deploymentCmd{
		RemoveAll: testCase.removeAll,
	}).RunRemoveDeployment(nil, testCase.args)

	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)

	err = os.Remove(constants.DefaultConfigPath)
	assert.Equal(t, !os.IsNotExist(err), testCase.expectConfigFile, "Unexpectedly saved or not saved in testCase %s", testCase.name)

}
