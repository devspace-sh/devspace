package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime/debug"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/mgutz/ansi"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type containerizeTestCase struct {
	name string

	files   map[string]interface{}
	answers []string

	pathFlag string

	expectedOutput string
	expectedPanic  string
}

func TestContainerize(t *testing.T) {
	testCases := []containerizeTestCase{
		containerizeTestCase{
			name:     "Dockerfile already exists",
			pathFlag: "Dockerfile",
			files: map[string]interface{}{
				"Dockerfile": "",
			},
			expectedPanic: "Error containerizing application: Dockerfile at Dockerfile already exists",
		},
		containerizeTestCase{
			name:           "Create new Dockerfile",
			pathFlag:       "Dockerfile",
			answers:        []string{"javascript"},
			expectedOutput: fmt.Sprintf("\nWait Detecting programming language\nInfo Successfully containerized project. Run: \n- `%s` to initialize DevSpace in the project\n- `%s` to verify that the Dockerfile is working in this project", ansi.Color("devspace init", "white+b"), ansi.Color("docker build .", "white+b")),
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testContainerize(t, testCase)
	}
}

func testContainerize(t *testing.T, testCase containerizeTestCase) {
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

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	(&ContainerizeCmd{
		Path: testCase.pathFlag,
	}).RunContainerize(nil, []string{})
}
