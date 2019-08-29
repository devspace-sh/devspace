package list

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

type listSelectorsTestCase struct {
	name string

	fakeConfig *latest.Config

	expectedOutput string
	expectedPanic  string
}

func TestListSelectors(t *testing.T) {
	expectedHeader := ansi.Color(" Name  ", "green+b") + "      " + ansi.Color(" Namespace  ", "green+b") + ansi.Color(" Label Selector  ", "green+b") + "          " + ansi.Color(" Container  ", "green+b") + "      "
	testCases := []listSelectorsTestCase{
		listSelectorsTestCase{
			name:          "no config exists",
			expectedPanic: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		listSelectorsTestCase{
			name: "no selectors exists",
			fakeConfig: &latest.Config{
				Dev: &latest.DevConfig{},
			},
			expectedOutput: "\nInfo No selectors are configured. Run `devspace add selector` to add new selector\n",
		},
		listSelectorsTestCase{
			name: "one selectors exists",
			fakeConfig: &latest.Config{
				Dev: &latest.DevConfig{
					Selectors: &[]*latest.SelectorConfig{
						&latest.SelectorConfig{
							Name: ptr.String("mySelector"),
							LabelSelector: &map[string]*string{
								"hello":  ptr.String("world"),
								"isTest": ptr.String("true"),
							},
							Namespace:     ptr.String("myNS"),
							ContainerName: ptr.String("myContainername"),
						},
					},
				},
			},
			expectedOutput: "\n" + expectedHeader + "\n mySelector   myNS        hello=world, isTest=true   myContainername  \n\n",
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testListSelectors(t, testCase)
	}
}

func testListSelectors(t *testing.T, testCase listSelectorsTestCase) {
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

	(&selectorsCmd{}).RunListSelectors(nil, []string{})

	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
