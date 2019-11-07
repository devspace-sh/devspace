package status

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"

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

func (t testLogger) Fail(args ...interface{}) {
	logOutput = logOutput + "\nFail " + fmt.Sprint(args...)
}
func (t testLogger) Failf(format string, args ...interface{}) {
	logOutput = logOutput + "\nFail " + fmt.Sprintf(format, args...)
}

func (t testLogger) Warn(args ...interface{}) {
	logOutput = logOutput + "\nWarn " + fmt.Sprint(args...)
}
func (t testLogger) Warnf(format string, args ...interface{}) {
	logOutput = logOutput + "\nWarn " + fmt.Sprintf(format, args...)
}

func (t testLogger) StartWait(msg string) {
	logOutput = logOutput + "\nWait " + fmt.Sprint(msg)
}

func (t testLogger) Write(msg []byte) (int, error) {
	logOutput = logOutput + string(msg)
	return len(msg), nil
}

type statusSyncTestCase struct {
	name string

	files map[string]interface{}

	expectedOutput string
	expectedErr    string
}

func TestRunStatusSync(t *testing.T) {
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

	expectedHeader := "\n" + ansi.Color(" Status  ", "green+b") + " " + ansi.Color(" Pod  ", "green+b") + "               " + ansi.Color(" Local  ", "green+b") + "                  " + ansi.Color(" Container  ", "green+b") + "              " + ansi.Color(" Latest Activity  ", "green+b") + "            " + ansi.Color(" Total Changes  ", "green+b")
	testCases := []statusSyncTestCase{
		/*statusSyncTestCase{
			name: "Empty sync.log",
			files: map[string]interface{}{
				constants.DefaultConfigPath: "",
				".devspace/logs/sync.log":   "",
			},
			expectedOutput: "\nInfo No sync activity found. Did you run `devspace dev`?",
		},
		statusSyncTestCase{
			name: "Valid sync.log",
			files: map[string]interface{}{
				constants.DefaultConfigPath: "",
				".devspace/logs/sync.log": []interface{}{
					map[string]string{
						"container": "someContainer",
						"local":     "someLocal",
						"pod":       "somePod",
						"level":     "error",
						"time":      "someTime",
						"msg":       "someMsg",
					},
					map[string]string{
						"container": "someContainer",
						"local":     "someLocal",
						"pod":       "somePod",
						"level":     "someLevel",
						"time":      time.Now().Add(-1 * time.Hour * 24).Format(time.RFC3339),
						"msg":       "[Downstream] Successfully processed 1 change(s)",
					},
					map[string]string{
						"container": "TooLongAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
						"local":     "TooLongAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
						"pod":       "TooLongAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
						"level":     "someLevel",
						"time":      time.Unix(0, 0).Format(time.RFC3339),
						"msg":       "[Upstream] Successfully processed 1 change(s)",
					},
					map[string]string{
						"container": "stoppedContainer",
						"local":     "stoppedLocal",
						"pod":       "stoppedPod",
						"level":     "someLevel",
						"time":      time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
						"msg":       "[Sync] Sync stopped",
					},
				},
			},
			expectedOutput: expectedHeader + "\n Active    TooLongAAAAAAAA...   ...AAAAAAAAAAAAAAAAAAAA   ...AAAAAAAAAAAAAAAAAAAA   Uploaded 1 changes (0s ago)   1              \n Stopped   stoppedPod           stoppedLocal              stoppedContainer          Sync stopped (1h ago)         0              \n Error     somePod              someLocal                 someContainer             someMsg (1d ago)              1              \n\n",
		},*/
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testRunStatusSync(t, testCase)
	}
}

func testRunStatusSync(t *testing.T, testCase statusSyncTestCase) {
	logOutput = ""

	for path, content := range testCase.files {
		asJSON, err := json.Marshal(content)
		assert.NilError(t, err, "Error parsing content to json in testCase %s", testCase.name)
		if content == "" {
			asJSON = []byte{}
		}
		if contentArr, ok := content.([]interface{}); ok {
			asJSON = []byte{}
			for _, contentToken := range contentArr {
				line, err := json.Marshal(contentToken)
				assert.NilError(t, err, "Error parsing content to json in testCase %s", testCase.name)
				asJSON = append(asJSON, line...)
				asJSON = append(asJSON, []byte("\n")...)
			}
		}
		err = fsutil.WriteToFile(asJSON, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	err := (&syncCmd{}).RunStatusSync(nil, []string{})

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
