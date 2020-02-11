package list

/*
import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"gotest.tools/assert"
)

type listPortsTestCase struct {
	name string

	fakeConfig *latest.Config

	expectTablePrint bool
	expectedHeader   []string
	expectedValues   [][]string
	expectedErr      string
}

func TestListPorts(t *testing.T) {
	testCases := []listPortsTestCase{
		listPortsTestCase{
			name: "two ports forwarded",
			fakeConfig: &latest.Config{
				Dev: &latest.DevConfig{
					Ports: []*latest.PortForwardingConfig{
						&latest.PortForwardingConfig{
							LabelSelector: map[string]string{
								"app": "test",
							},
							PortMappings: []*latest.PortMapping{
								&latest.PortMapping{
									LocalPort:  ptr.Int(1234),
									RemotePort: ptr.Int(4321),
								},
								&latest.PortMapping{
									LocalPort:  ptr.Int(5678),
									RemotePort: ptr.Int(8765),
								},
							},
						},
						&latest.PortForwardingConfig{
							//The order can be any way, so we do a little trick so the selectors are printed equally
							LabelSelector: map[string]string{
								"a":   "b=",
								"a=b": "",
							},
							PortMappings: []*latest.PortMapping{
								&latest.PortMapping{
									LocalPort:  ptr.Int(9012),
									RemotePort: ptr.Int(2109),
								},
							},
						},
					},
				},
			},
			expectedHeader: []string{"Image", "LabelSelector", "Ports (Local:Remote)"},
			expectedValues: [][]string{
				[]string{"", "app=test", "1234:4321, 5678:8765"},
				[]string{"", "a=b=, a=b=", "9012:2109"},
			},
		},
	}

	log.SetInstance(log.Discard)

	for _, testCase := range testCases {
		testListPorts(t, testCase)
	}
}

func testListPorts(t *testing.T, testCase listPortsTestCase) {
	log.SetFakePrintTable(func(s log.Logger, header []string, values [][]string) {
		assert.Assert(t, testCase.expectTablePrint || len(testCase.expectedHeader)+len(testCase.expectedValues) > 0, "PrintTable unexpectedly called in testCase %s", testCase.name)
		assert.Equal(t, reflect.DeepEqual(header, testCase.expectedHeader), true, "Unexpected header in testCase %s. Expected:%v\nActual:%v", testCase.name, testCase.expectedHeader, header)
		assert.Equal(t, reflect.DeepEqual(values, testCase.expectedValues), true, "Unexpected values in testCase %s. Expected:%v\nActual:%v", testCase.name, testCase.expectedValues, values)
	})

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

	loader.SetFakeConfig(testCase.fakeConfig)

	err = (&portsCmd{GlobalFlags: &flags.GlobalFlags{}}).RunListPort(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}*/
