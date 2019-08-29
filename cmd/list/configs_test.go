package list

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime/debug"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configs"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/mgutz/ansi"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type listConfigsTestCase struct {
	name string

	fakeConfig           *latest.Config
	configsYamlContent   interface{}
	generatedYamlContent interface{}

	expectedOutput string
	expectedPanic  string
}

func TestListConfigs(t *testing.T) {
	expectedHeader := ansi.Color(" Name  ", "green+b") + ansi.Color(" Active  ", "green+b") + ansi.Color(" Path  ", "green+b") + ansi.Color(" Vars  ", "green+b") + ansi.Color(" Overwrites  ", "green+b")
	testCases := []listConfigsTestCase{
		listConfigsTestCase{
			name:          "no config exists",
			expectedPanic: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		listConfigsTestCase{
			name:           "no configs.yaml exists",
			fakeConfig:     &latest.Config{},
			expectedOutput: fmt.Sprintf("\nInfo Please create a '%s' to specify multiple configurations", constants.DefaultConfigsPath),
		},
		listConfigsTestCase{
			name:               "configs.yaml has unparsable content",
			fakeConfig:         &latest.Config{},
			configsYamlContent: "unparsable",
			expectedPanic:      fmt.Sprintf("Error loading %s: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into configs.Configs", constants.DefaultConfigsPath),
		},
		listConfigsTestCase{
			name:                 "generated.yaml has unparsable content",
			fakeConfig:           &latest.Config{},
			configsYamlContent:   configs.Configs{},
			generatedYamlContent: "unparsable",
			expectedPanic:        "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into generated.Config",
		},
		listConfigsTestCase{
			name:               "no configs available",
			fakeConfig:         &latest.Config{},
			configsYamlContent: configs.Configs{},
			expectedOutput:     "\n" + expectedHeader + "\n No entries found\n\n",
		},
		listConfigsTestCase{
			name:       "one config",
			fakeConfig: &latest.Config{},
			configsYamlContent: configs.Configs{
				"cnf": &configs.ConfigDefinition{
					Config: &configs.ConfigWrapper{
						Path: ptr.String("pth"),
					},
					Overrides: &[]*configs.ConfigWrapper{&configs.ConfigWrapper{}},
					Vars:      map[string]string{},
				},
			},
			expectedOutput: "\n" + expectedHeader + "\n cnf    false    pth    true   1           \n\n",
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testListConfigs(t, testCase)
	}
}

func testListConfigs(t *testing.T, testCase listConfigsTestCase) {
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
	generated.ResetConfig()

	if testCase.configsYamlContent != nil {
		content, err := yaml.Marshal(testCase.configsYamlContent)
		assert.NilError(t, err, "Error parsing configs.yaml to yaml in testCase %s", testCase.name)
		fsutil.WriteToFile(content, constants.DefaultConfigsPath)
	}

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

	(&configsCmd{}).RunListConfigs(nil, []string{})

	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
