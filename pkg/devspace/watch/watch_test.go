package watch

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	
	"gotest.tools/assert"
)

func TestWatcher(t *testing.T) {
	t.Skip("Travis blocks because of a data race.")
	//Create TmpFolder
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
	defer os.Chdir(wdBackup)
	defer os.RemoveAll(dir)

	callbackCalledChan := make(chan bool)
	expectedChanges := &[]string{}
	expectedDeletions := &[]string{}

	callback := func(changed []string, deleted []string) error{
		assert.Equal(t, len(*expectedChanges), len(changed), "Wrong changes")
		for index := range changed{
			assert.Equal(t, (*expectedChanges)[index], changed[index], "Wrong changes")
		}

		assert.Equal(t, len(*expectedDeletions), len(deleted), "Wrong deletions")
		for index := range deleted{
			assert.Equal(t, (*expectedDeletions)[index], deleted[index], "Wrong deletions")
		}

		callbackCalledChan <- true
		return nil
	}

	watcher, err := New([]string{".", "hello.txt", "watchedsubdir"}, callback, log.GetInstance())
	if err != nil {
		t.Fatalf("Error creating watcher: %v", err)
	}

	watcher.Start()

	*expectedChanges = []string{".", "hello.txt"}
	fsutil.WriteToFile([]byte("hello"), "hello.txt")
	select{
		case <- callbackCalledChan:
		case <- time.After(time.Second * 5):
			t.Fatalf("Timeout of waiting for callback after creating a file")
	}
	
	*expectedChanges = []string{".", "watchedsubdir"}
	fsutil.WriteToFile([]byte("hi"), "watchedsubdir/unwatchedsubfile.txt")
	select{
		case <- callbackCalledChan:
		case <- time.After(time.Second * 5):
			t.Fatalf("Timeout of waiting for callback after creating a secound file")
	}

	*expectedChanges = []string{"hello.txt"}
	err = fsutil.WriteToFile([]byte("another hello"), "hello.txt")
	if err != nil {
		t.Fatalf("Error changing file: %v", err)
	}
	select{
		case <- callbackCalledChan:
		case <- time.After(time.Second * 5):
			t.Fatalf("Timeout of waiting for callback after changing a file")
	}

	*expectedChanges = []string{"."}
	*expectedDeletions = []string{"hello.txt"}
	err = os.Remove("hello.txt")
	if err != nil {
		t.Fatalf("Error deleting file: %v", err)
	}
	select{
		case <- callbackCalledChan:
		case <- time.After(time.Second * 5):
			t.Fatalf("Timeout of waiting for callback after changing a file")
	}

	watcher.Stop()

}
