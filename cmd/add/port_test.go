package add

/*
import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"gotest.tools/assert"
)

type addPortTestCase struct {
	name string

	args       []string
	fakeConfig *latest.Config
	cmd        *portCmd

	expectedErr      string
	expectConfigFile bool
	expectedPorts    []*latest.PortMapping
}

func TestRunAddPort(t *testing.T) {
	testCases := []addPortTestCase{
		addPortTestCase{
			name:             "Add valid port",
			args:             []string{"1234"},
			fakeConfig:       &latest.Config{},
			expectConfigFile: true,
			expectedPorts: []*latest.PortMapping{
				&latest.PortMapping{
					LocalPort: ptr.Int(1234),
				},
			},
		},
	}

	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testRunAddPort(t, testCase)
	}
}

func testRunAddPort(t *testing.T, testCase addPortTestCase) {
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

	isDeploymentsNil := testCase.fakeConfig == nil || testCase.fakeConfig.Deployments == nil
	loader.SetFakeConfig(testCase.fakeConfig)
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
	}()

	if testCase.cmd == nil {
		testCase.cmd = &portCmd{}
	}
	if testCase.cmd.GlobalFlags == nil {
		testCase.cmd.GlobalFlags = &flags.GlobalFlags{}
	}

	err = (testCase.cmd).RunAddPort(nil, testCase.args)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
		return
	}

	config, err := loader.GetBaseConfig(&loader.ConfigOptions{})
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, len(testCase.expectedPorts), len(config.Dev.Ports[0].PortMappings), "Wrong number of port mappings in testCase %s", testCase.name)
	for index, portMapping := range config.Dev.Ports[0].PortMappings {
		assert.Equal(t, *testCase.expectedPorts[index].LocalPort, *portMapping.LocalPort, "Local port unexpected in testCase %s", testCase.name)
	}

	err = os.Remove(constants.DefaultConfigPath)
	assert.Equal(t, !os.IsNotExist(err), testCase.expectConfigFile, "Unexpectedly saved or not saved in testCase %s", testCase.name)
}*/
