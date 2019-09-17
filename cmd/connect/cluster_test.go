package connect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"gotest.tools/assert"
)

var logOutput string

type testLogger struct {
	log.DiscardLogger
}

func (t testLogger) Info(args ...interface{}) {
	logOutput = logOutput + "\nInfo " + fmt.Sprint(args...)
}
func (t testLogger) Infof(format string, args ...interface{}) {
	logOutput = logOutput + "\nInfo " + fmt.Sprintf(format, args...)
}

func (t testLogger) Done(args ...interface{}) {
	logOutput = logOutput + "\nDone " + fmt.Sprint(args...)
}
func (t testLogger) Donef(format string, args ...interface{}) {
	logOutput = logOutput + "\nDone " + fmt.Sprintf(format, args...)
}

func (t testLogger) Warn(args ...interface{}) {
	logOutput = logOutput + "\nWarn " + fmt.Sprint(args...)
}
func (t testLogger) Warnf(format string, args ...interface{}) {
	logOutput = logOutput + "\nWarn " + fmt.Sprintf(format, args...)
}

func (t testLogger) StartWait(message string) {
	logOutput = logOutput + "\nWait " + message
}

type customGraphqlClient struct {
	responses []interface{}
}

func (q *customGraphqlClient) GrapqhlRequest(p *cloudpkg.Provider, request string, vars map[string]interface{}, response interface{}) error {
	if len(q.responses) == 0 {
		panic("Not enough responses. Need response for: " + request)
	}
	currentResponse := q.responses[0]
	q.responses = q.responses[1:]

	errorResponse, isError := currentResponse.(error)
	if isError {
		return errorResponse
	}
	buf, err := json.Marshal(currentResponse)
	if err != nil {
		panic(fmt.Sprintf("Cannot encode response. %d responses left", len(q.responses)))
	}
	json.NewDecoder(bytes.NewReader(buf)).Decode(&response)

	return nil
}

type connectClusterTestCase struct {
	name string

	graphQLResponses []interface{}
	providerList     []*cloudlatest.Provider

	providerFlag       string
	useHostNetworkFlag bool
	optionsFlag        *cloudpkg.ConnectClusterOptions

	expectedOutput string
	expectedErr    string
}

func TestRunConnectCluster(t *testing.T) {
	testCases := []connectClusterTestCase{
		connectClusterTestCase{
			name:         "Provider doesn't Exist",
			providerFlag: "Doesn'tExist",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name: "SomeProvider",
				},
			},
			expectedErr: "Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: SomeProvider ",
		},
		connectClusterTestCase{
			name:         "Invalid cluster name",
			providerFlag: "SomeProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name: "SomeProvider",
					Key:  "someKey",
				},
			},
			graphQLResponses: []interface{}{
				fmt.Errorf("Custom server error"),
			},
			useHostNetworkFlag: true,
			optionsFlag: &cloudpkg.ConnectClusterOptions{
				ClusterName: "!nva|id clu5ter_nam3",
			},
			expectedErr: "Cluster name !nva|id clu5ter_nam3 can only contain letters, numbers and dashes (-)",
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testRunConnectCluster(t, testCase)
	}
}

func testRunConnectCluster(t *testing.T, testCase connectClusterTestCase) {
	logOutput = ""

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

		assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
	}()

	cloudpkg.DefaultGraphqlClient = &customGraphqlClient{
		responses: testCase.graphQLResponses,
	}

	providerConfig, err := cloudconfig.ParseProviderConfig()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	cobraCmd := newClusterCmd()
	cobraCmd.Flag("use-hostnetwork").Changed = true
	err = (&clusterCmd{
		Provider:       testCase.providerFlag,
		UseHostNetwork: testCase.useHostNetworkFlag,
		Options:        testCase.optionsFlag,
	}).RunConnectCluster(cobraCmd, nil)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}
