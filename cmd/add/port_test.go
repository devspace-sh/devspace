package add

import (
	"context"
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

type addPortTestCase struct {
	name string

	args          []string
	answers       []string
	fakeConfig    *latest.Config
	labelSelector string
	namespace     string
	service       string

	expectedOutput   string
	expectedPanic    string
	expectConfigFile bool
	expectedPorts    []*latest.PortMapping
}

func TestRunAddPort(t *testing.T) {
	testCases := []addPortTestCase{
		addPortTestCase{
			name:          "No devspace config",
			args:          []string{""},
			expectedPanic: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		addPortTestCase{
			name:          "Add empty port",
			args:          []string{""},
			fakeConfig:    &latest.Config{},
			expectedPanic: "Error parsing port mappings: strconv.Atoi: parsing \"\": invalid syntax",
		},
		addPortTestCase{
			name:             "Add valid port",
			args:             []string{"1234"},
			fakeConfig:       &latest.Config{},
			expectedOutput:   "\nDone Successfully added port 1234",
			expectConfigFile: true,
			expectedPorts: []*latest.PortMapping{
				&latest.PortMapping{
					LocalPort: ptr.Int(1234),
				},
			},
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testRunAddPort(t, testCase)
	}
}

func testRunAddPort(t *testing.T, testCase addPortTestCase) {
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

	(&portCmd{
		LabelSelector: testCase.labelSelector,
		Namespace:     testCase.namespace,
		Service:       testCase.service,
	}).RunAddPort(nil, testCase.args)

	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)

	config := configutil.GetBaseConfig(context.Background())

	assert.Equal(t, len(testCase.expectedPorts), len(config.Dev.Ports[0].PortMappings), "Wrong number of port mappings in testCase %s", testCase.name)
	for index, portMapping := range config.Dev.Ports[0].PortMappings {
		assert.Equal(t, *testCase.expectedPorts[index].LocalPort, *portMapping.LocalPort, "Local port unexpected in testCase %s", testCase.name)
	}

	err = os.Remove(constants.DefaultConfigPath)
	assert.Equal(t, !os.IsNotExist(err), testCase.expectConfigFile, "Unexpectedly saved or not saved in testCase %s", testCase.name)
}
