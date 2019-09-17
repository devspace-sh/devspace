package list

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"

	"gotest.tools/assert"
)

type listSyncsTestCase struct {
	name string

	fakeConfig *latest.Config

	expectedOutput string
	expectedErr    string
}

func TestListSyncs(t *testing.T) {
	expectedHeader := ansi.Color(" Label Selector  ", "green+b") + ansi.Color(" Local Path  ", "green+b") + ansi.Color(" Container Path  ", "green+b") + ansi.Color(" Excluded Paths  ", "green+b")
	testCases := []listSyncsTestCase{
		listSyncsTestCase{
			name:        "no config exists",
			expectedErr: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		listSyncsTestCase{
			name: "no sync paths exists",
			fakeConfig: &latest.Config{
				Dev: &latest.DevConfig{},
			},
			expectedOutput: "\nInfo No sync paths are configured. Run `devspace add sync` to add new sync path\n",
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
			expectedOutput: "\n" + expectedHeader + "\n app=test         local        container        path1, path2    \n a=b=, a=b=       local2       container2                       \n\n",
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testListSyncs(t, testCase)
	}
}

func testListSyncs(t *testing.T, testCase listSyncsTestCase) {
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

	err = (&syncCmd{GlobalFlags: &flags.GlobalFlags{}}).RunListSync(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
