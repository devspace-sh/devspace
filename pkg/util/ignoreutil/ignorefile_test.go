package ignoreutil

import (
	"io/ioutil"
	"os"
	"testing"
	
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	
	"gotest.tools/assert"
)

func TestGetIgnoreRules(t *testing.T){
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

	// 8. Delete temp folder
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

	fsutil.WriteToFile([]byte("notignore"), "NotDockerIgnore")
	fsutil.WriteToFile([]byte(`ignoreFile`), ".dockerignore")
	fsutil.WriteToFile([]byte(`ignoreFile`), "someDir/.dockerignore")

	ignoreRules, err := GetIgnoreRules(".")
	if err != nil {
		t.Fatalf("Error getting ignoreRules: %v", err)
	}
	assert.Equal(t, 2, len(ignoreRules), "Wrong number of ignoreRules")
	assert.Equal(t, true, contains(ignoreRules, "ignoreFile"))
	assert.Equal(t, true, contains(ignoreRules, "someDir/**/ignoreFile"))
}

func contains(s []string, e string) bool {
    for _, a := range s {
        if a == e {
            return true
        }
    }
    return false
}
