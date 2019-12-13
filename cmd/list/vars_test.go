package list

/*
import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"gopkg.in/yaml.v2"

	"gotest.tools/assert"
)

type listVarsTestCase struct {
	name string

	fakeConfig           *latest.Config
	generatedYamlContent interface{}

	expectTablePrint bool
	expectedHeader   []string
	expectedValues   [][]string
	expectedErr      string
}

func TestListVars(t *testing.T) {
	testCases := []listVarsTestCase{
		listVarsTestCase{
			name:       "no vars",
			fakeConfig: &latest.Config{},
		},
		listVarsTestCase{
			name:       "one var",
			fakeConfig: &latest.Config{},
			generatedYamlContent: generated.Config{
				ActiveProfile: "myConf",
				Profiles: map[string]*generated.CacheConfig{
					"myConf": &generated.CacheConfig{},
				},
				Vars: map[string]string{
					"hello": "world",
				},
			},
			expectedHeader: []string{"Variable", "Value"},
			expectedValues: [][]string{
				[]string{"hello", "world"},
			},
		},
	}

	log.SetInstance(log.Discard)

	for _, testCase := range testCases {
		testListVars(t, testCase)
	}
}

func testListVars(t *testing.T, testCase listVarsTestCase) {
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
	generated.ResetConfig()

	if testCase.generatedYamlContent != nil {
		content, err := yaml.Marshal(testCase.generatedYamlContent)
		assert.NilError(t, err, "Error parsing configs.yaml to yaml in testCase %s", testCase.name)
		fsutil.WriteToFile(content, generated.ConfigPath)
	}

	err = (&varsCmd{GlobalFlags: &flags.GlobalFlags{}}).RunListVars(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}*/
