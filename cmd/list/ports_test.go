package list

/* @Florian adjust to new behaviour
import (
	"io/ioutil"
	"os"
	"runtime/debug"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/mgutz/ansi"

	"gotest.tools/assert"
)

type listPortsTestCase struct {
	name string

	fakeConfig *latest.Config

	expectedOutput string
	expectedPanic  string
}

func TestListPorts(t *testing.T) {
	expectedHeader := ansi.Color(" Selector  ", "green+b") + "  " + ansi.Color(" LabelSelector  ", "green+b") + ansi.Color(" Ports (Local:Remote)  ", "green+b")
	testCases := []listPortsTestCase{
		listPortsTestCase{
			name:          "no config exists",
			expectedPanic: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		listPortsTestCase{
			name: "no ports forwarded",
			fakeConfig: &latest.Config{
				Dev: &latest.DevConfig{},
			},
			expectedOutput: "\nInfo No ports are forwarded. Run `devspace add port` to add a port that should be forwarded\n",
		},
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
			expectedOutput: "\n" + expectedHeader + "\n mySelector                   1234:4321, 5678:8765  \n              a=b=, a=b=      9012:2109             \n\n",
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testListPorts(t, testCase)
	}
}

func testListPorts(t *testing.T, testCase listPortsTestCase) {
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

	configutil.SetFakeConfig(testCase.fakeConfig)

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

	(&portsCmd{}).RunListPort(nil, []string{})

	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
*/
