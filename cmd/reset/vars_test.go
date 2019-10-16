package reset

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type resetVarsTestCase struct {
	name string

	files map[string]interface{}

	expectedOutput string
	expectedErr    string
}

func TestRunResetVars(t *testing.T) {
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
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
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

	fsutil.WriteToFile([]byte(""), "someFakeDir")
	err = fsutil.WriteToFile([]byte(""), "someFakeDir/someFile")
	parentDirIsFileErr := strings.TrimPrefix(err.Error(), "mkdir someFakeDir: ")

	testCases := []resetVarsTestCase{
		resetVarsTestCase{
			name:        "No devspace.yaml",
			expectedErr: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		resetVarsTestCase{
			name: "Unparsable generated.yaml",
			files: map[string]interface{}{
				constants.DefaultConfigPath: "",
				".devspace/generated.yaml":  "unparsable",
			},
			expectedErr: "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `unparsable` into generated.Config",
		},
		resetVarsTestCase{
			name: "Unsavable generated.yaml",
			files: map[string]interface{}{
				constants.DefaultConfigPath: "",
				".devspace":                 "",
			},
			expectedErr: fmt.Sprintf("Error saving config: mkdir %s: %s", filepath.Join(dir, ".devspace"), parentDirIsFileErr),
		},
		resetVarsTestCase{
			name: "Success",
			files: map[string]interface{}{
				constants.DefaultConfigPath: "",
			},
			expectedOutput: "\nDone Successfully deleted all variables",
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testRunResetVars(t, testCase)
	}
}

func testRunResetVars(t *testing.T, testCase resetVarsTestCase) {
	logOutput = ""
	generated.ResetConfig()

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	err := (&varsCmd{}).RunResetVars(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)

	err = filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
		os.RemoveAll(path)
		return nil
	})
	assert.NilError(t, err, "Error cleaning up in testCase %s", testCase.name)
}
