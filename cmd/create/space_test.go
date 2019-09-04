package create

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
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

type createSpaceTestCase struct {
	name string

	graphQLResponses []interface{}
	providerList     []*cloudlatest.Provider

	activeFlag   bool
	providerFlag string
	clusterFlag  string

	expectedOutput string
	expectedPanic  string
}

func TestRunCreateSpace(t *testing.T) {
	testCases := []createSpaceTestCase{
		createSpaceTestCase{
			name:         "Provider doesn't Exist",
			providerFlag: "Doesn'tExist",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name: "SomeProvider",
				},
			},
			expectedPanic: "Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: SomeProvider ",
		},
		createSpaceTestCase{
			name:         "Projects can't be retrieved",
			providerFlag: "SomeProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name: "SomeProvider",
					Key:  "someKey",
				},
			},
			graphQLResponses: []interface{}{
				fmt.Errorf("Custom graphQL server error"),
			},
			expectedPanic:  "Error retrieving projects: Custom graphQL server error",
			expectedOutput: "\nWait Retrieving clusters",
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testRunCreateSpace(t, testCase)
	}
}

func testRunCreateSpace(t *testing.T, testCase createSpaceTestCase) {
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

	cloudpkg.DefaultGraphqlClient = &customGraphqlClient{
		responses: testCase.graphQLResponses,
	}

	providerConfig, err := cloudconfig.ParseProviderConfig()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	(&spaceCmd{
		Active:   testCase.activeFlag,
		Provider: testCase.providerFlag,
		Cluster:  testCase.clusterFlag,
	}).RunCreateSpace(nil, nil)
}
