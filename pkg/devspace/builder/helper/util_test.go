package helper

import (
	"io/ioutil"
	"os"
	"runtime"
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

	_, err = CreateTempDockerfile("", nil)
	assert.Error(t, err, "Entrypoint is empty", "Wrong error or wrong error returned when trying to create a temporary Dockerfile with nil entrypoints")
	_, err = CreateTempDockerfile("", []string{})
	assert.Error(t, err, "Entrypoint is empty", "Wrong error or wrong error returned when trying to create a temporary Dockerfile with 0 entrypoints")
	_, err = CreateTempDockerfile("", []string{""})
	assert.Error(t, err, "Entrypoint is empty", "Wrong error or wrong error returned when trying to create a temporary Dockerfile with only nil entrypoints")

	expectedErrorString := "open Doesn'tExist: The system cannot find the file specified."
	if runtime.GOOS != "windows" {
		expectedErrorString = "open Doesn'tExist: no such file or directory"
	}
	_, err = CreateTempDockerfile("Doesn'tExist", []string{"echo"})
	assert.Error(t, err, expectedErrorString, "Wrong or no error when trying to create a dockerfile from an non existent dockerfile")

	err = fsutil.WriteToFile([]byte(""), "Exists")
	dockerfilepath, err := CreateTempDockerfile("Exists", []string{"echo"})
	assert.NilError(t, err, "Error when creating a valid temporary Dockerfile")
	dockerfileContent, err := fsutil.ReadFile(dockerfilepath, -1)
	assert.NilError(t, err, "Temporary Dockerfile not created.")
	assert.Equal(t, `

ENTRYPOINT ["echo"]
CMD [""]`, string(dockerfileContent), "Temporary dockerfile has wrong content")
}
