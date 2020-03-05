package status

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	fakeloader "github.com/devspace-cloud/devspace/pkg/devspace/config/loader/testing"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	fakefactory "github.com/devspace-cloud/devspace/pkg/util/factory/testing"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"

	"gotest.tools/assert"
)

type statusSyncTestCase struct {
	name string

	files  map[string]interface{}
	config *latest.Config

	expectedErr bool
}

type logentry struct {
	Level   string `json:"level"`
	Message string `json:"msg"`
	Time    string `json:"time"`
}

func TestRunStatusSync(t *testing.T) {
	testCases := []statusSyncTestCase{
		statusSyncTestCase{
			name: "No devspace.yaml",
			files: map[string]interface{}{
				".devspace/logs/sync.log": "",
			},
			expectedErr: true,
		},
		statusSyncTestCase{
			name:        "No sync.log",
			config:      &latest.Config{},
			expectedErr: true,
		},
		statusSyncTestCase{
			name: "Empty sync.log",
			files: map[string]interface{}{
				".devspace/logs/sync.log": "",
			},
			config: &latest.Config{},
		},
		statusSyncTestCase{
			name: "Uncomplete json line in sync.log",
			files: map[string]interface{}{
				".devspace/logs/sync.log": []interface{}{logentry{}},
			},
			config:      &latest.Config{},
			expectedErr: true,
		},
		statusSyncTestCase{
			name: "Complete json line in sync.log",
			files: map[string]interface{}{
				".devspace/logs/sync.log": []interface{}{
					logentry{
						Level:   "mylevel",
						Message: "msg",
						Time:    time.Now().String(),
					},
				},
			},
			config: &latest.Config{},
		},
	}

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
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	for _, testCase := range testCases {
		testRunStatusSync(t, testCase)
	}
}

func testRunStatusSync(t *testing.T, testCase statusSyncTestCase) {
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

	factory := &fakefactory.Factory{
		ConfigLoader: &fakeloader.FakeConfigLoader{
			Config: testCase.config,
		},
		Log: &log.FakeLogger{},
	}

	err := (&syncCmd{}).RunStatusSync(factory, nil, []string{})

	if !testCase.expectedErr {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else if err == nil {
		t.Fatalf("Unexpeted no error in testCase %s", testCase.name)
	}

	err = filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
		os.RemoveAll(path)
		return nil
	})
	assert.NilError(t, err, "Error cleaning up in testCase %s", testCase.name)
}
