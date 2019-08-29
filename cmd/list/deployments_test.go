package list

import (
	"io/ioutil"
	"os"
	"runtime/debug"
	"testing"

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

type listDeploymentsTestCase struct {
	name string

	fakeConfig           *latest.Config
	configsYamlContent   interface{}
	generatedYamlContent interface{}
	providerList         []*cloudlatest.Provider

	expectedOutput string
	expectedPanic  string
}

func TestListDeployments(t *testing.T) {
	//expectedHeader := ansi.Color(" NAME  ", "green+b") + ansi.Color(" TYPE  ", "green+b") + ansi.Color(" DEPLOY  ", "green+b") + ansi.Color(" STATUS  ", "green+b")
	testCases := []listDeploymentsTestCase{
		listDeploymentsTestCase{
			name:          "no config exists",
			expectedPanic: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		listDeploymentsTestCase{
			name:                 "generated.yaml has unparsable content",
			fakeConfig:           &latest.Config{},
			generatedYamlContent: "unparsable",
			expectedPanic:        "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into generated.Config",
		},
		listDeploymentsTestCase{
			name:       "Space can't be resumed",
			fakeConfig: &latest.Config{},
			generatedYamlContent: generated.Config{
				CloudSpace: &generated.CloudSpaceConfig{},
			},
			expectedPanic: "Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: ",
		},
		/*listDeploymentsTestCase{
			name:           "No deployments",
			fakeConfig:     &latest.Config{},
			expectedOutput: "\n" + expectedHeader + "\n No entries found\n\n",
		},
		listDeploymentsTestCase{
			name: "Print deployments",
			fakeConfig: &latest.Config{
				Cluster: &latest.Cluster{Namespace: ptr.String("someNS")},
				Deployments: &[]*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name:    ptr.String("UndeployableKubectl"),
						Kubectl: &latest.KubectlConfig{},
					},
					&latest.DeploymentConfig{
						Name: ptr.String("NoDeploymentMethod"),
					},
				},
			},
			expectedOutput: "\nWarn Unable to create kubectl deploy config for UndeployableKubectl: No manifests defined for kubectl deploy\nWarn No deployment method defined for deployment NoDeploymentMethod" + "\n" + expectedHeader + "\n No entries found\n\n",
		},*/
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testListDeployments(t, testCase)
	}
}

func testListDeployments(t *testing.T, testCase listDeploymentsTestCase) {
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

	providerConfig, err := cloudconfig.ParseProviderConfig()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	configutil.SetFakeConfig(testCase.fakeConfig)
	generated.ResetConfig()

	if testCase.generatedYamlContent != nil {
		content, err := yaml.Marshal(testCase.generatedYamlContent)
		assert.NilError(t, err, "Error parsing configs.yaml to yaml in testCase %s", testCase.name)
		fsutil.WriteToFile(content, generated.ConfigPath)
	}

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
	}()

	(&deploymentsCmd{}).RunDeploymentsStatus(nil, []string{})

	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
