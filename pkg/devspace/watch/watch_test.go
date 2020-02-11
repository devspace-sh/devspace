package watch

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"

	"gotest.tools/assert"
)

type testCase struct {
	name              string
	changes           []testCaseChange
	expectedChanges   []string
	expectedDeletions []string
}

type testCaseChange struct {
	path    string
	content string
	delete  bool
}

func TestWatcher(t *testing.T) {
	watchedPaths := []string{".", "hello.txt", "watchedsubdir"}
	testCases := []testCase{
		{
			name:            "Create text file",
			expectedChanges: []string{"hello.txt"},
			changes: []testCaseChange{
				{
					path:    "hello.txt",
					content: "hello",
				},
			},
		},
		{
			name:            "Create file in folder",
			expectedChanges: []string{"watchedsubdir"},
			changes: []testCaseChange{
				{
					path:    "watchedsubdir/unwatchedsubfile.txt",
					content: "watchedsubdir",
				},
			},
		},
		{
			name:            "Override file",
			expectedChanges: []string{"hello.txt"},
			changes: []testCaseChange{
				{
					path:    "hello.txt",
					content: "another hello",
				},
			},
		},
		{
			name:              "Delete file",
			expectedChanges:   []string{},
			expectedDeletions: []string{"hello.txt"},
			changes: []testCaseChange{
				{
					path:   "hello.txt",
					delete: true,
				},
			},
		},
	}

	// Create TmpFolder
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

	// Cleanup temp folder
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

	var (
		callbackCalledChan = make(chan error)
		expectedChanges    = &[]string{}
		expectedDeletions  = &[]string{}
		changeLock         sync.Mutex
	)

	callback := func(changed []string, deleted []string) error {
		changeLock.Lock()
		defer changeLock.Unlock()

		for _, change := range changed {
			indexInExpected := indexOf(change, *expectedChanges)
			if indexInExpected == -1 {
				callbackCalledChan <- errors.Errorf("Unexpected change in %s", change)
				return nil
			}
			*expectedChanges = append((*expectedChanges)[:indexInExpected], (*expectedChanges)[indexInExpected+1:]...)
		}

		for _, deletion := range deleted {
			indexInExpected := indexOf(deletion, *expectedDeletions)
			if indexInExpected == -1 {
				callbackCalledChan <- errors.Errorf("Unexpected deletion of %s", deletion)
				return nil
			}
			*expectedDeletions = append((*expectedDeletions)[:indexInExpected], (*expectedDeletions)[indexInExpected+1:]...)
		}

		if len(*expectedChanges) == 0 && len(*expectedDeletions) == 0 {
			callbackCalledChan <- nil
		}
		return nil
	}

	watcherObj, err := New(watchedPaths, []string{}, time.Millisecond * 10, callback, log.GetInstance())
	if err != nil {
		t.Fatalf("Error creating watcher: %v", err)
	}

	watcherObj.Start()

	for _, testCase := range testCases {
		changeLock.Lock()
		if testCase.expectedChanges != nil {
			*expectedChanges = testCase.expectedChanges
		} else {
			*expectedChanges = []string{}
		}
		if testCase.expectedDeletions != nil {
			*expectedDeletions = testCase.expectedDeletions
		} else {
			*expectedDeletions = []string{}
		}
		changeLock.Unlock()

		// Apply changes
		for _, change := range testCase.changes {
			if change.delete {
				err = os.Remove(change.path)
				if err != nil {
					t.Fatalf("Error deleting file %s: %v", change.path, err)
				}
			} else {
				err = fsutil.WriteToFile([]byte(change.content), change.path)
				if err != nil {
					t.Fatalf("Error creating file %s: %v", change.path, err)
				}
			}
		}

		select {
		case err = <-callbackCalledChan:
			assert.NilError(t, err, "Test %s failed", testCase.name)
		case <-time.After(time.Second * 5):
			changeLock.Lock()
			t.Fatalf("Test %s timed out. Remaining changes: %v . Remaining deletions: %v", testCase.name, *expectedChanges, *expectedDeletions)
		}
	}

	watcherObj.Stop()
}

func indexOf(element string, data []string) int {
	for k, v := range data {
		if element == v {
			return k
		}
	}
	return -1 //not found.
}
