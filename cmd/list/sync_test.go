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

	"gotest.tools/assert"
)

type listSyncsTestCase struct {
	name string

	fakeConfig *latest.Config

	expectTablePrint bool
	expectedHeader   []string
	expectedValues   [][]string
	expectedErr      string
}

func TestListSyncs(t *testing.T) {
	testCases := []listSyncsTestCase{
		listSyncsTestCase{
			name: "no sync paths exists",
			fakeConfig: &latest.Config{
				Dev: &latest.DevConfig{},
			},
		},
		listSyncsTestCase{
			name: "Print one sync path",
			fakeConfig: &latest.Config{
				Dev: &latest.DevConfig{
					Sync: []*latest.SyncConfig{
						&latest.SyncConfig{
							LocalSubPath:  "local",
							ContainerPath: "container",
							LabelSelector: map[string]string{
								"app": "test",
							},
							ExcludePaths: []string{"path1", "path2"},
						},
						&latest.SyncConfig{
							LocalSubPath:  "local2",
							ContainerPath: "container2",
							LabelSelector: map[string]string{
								//The order can be any way, so we do a little trick so the selectors are printed equally
								"a":   "b=",
								"a=b": "",
							},
						},
					},
				},
			},
			expectedHeader: []string{"Label Selector", "Local Path", "Container Path", "Excluded Paths"},
			expectedValues: [][]string{
				[]string{"app=test", "local", "container", "path1, path2"},
				[]string{"a=b=, a=b=", "local2", "container2", ""},
			},
		},
	}

	log.SetInstance(log.Discard)

	for _, testCase := range testCases {
		testListSyncs(t, testCase)
	}
}

func testListSyncs(t *testing.T, testCase listSyncsTestCase) {
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

	err = (&syncCmd{GlobalFlags: &flags.GlobalFlags{}}).RunListSync(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}*/
