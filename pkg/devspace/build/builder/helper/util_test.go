package helper

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/util/fsutil"

	"gotest.tools/assert"
)

func TestCreateTempDockerfile(t *testing.T) {
	//Create tempDir and go into it
	dir, err := ioutil.TempDir("", "testDir")
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

	// Delete temp folder after test
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

	err = fsutil.WriteToFile([]byte(""), "Exists")
	dockerfilepath, err := CreateTempDockerfile("Exists", []string{"echo"}, []string{""}, "")
	assert.NilError(t, err, "Error when creating a valid temporary Dockerfile")
	dockerfileContent, err := fsutil.ReadFile(dockerfilepath, -1)
	assert.NilError(t, err, "Temporary Dockerfile not created.")
	assert.Equal(t, "\n\nENTRYPOINT [\"echo\"]\n\n\nCMD [\"\"]\n", string(dockerfileContent), "Temporary dockerfile has wrong content")
}
