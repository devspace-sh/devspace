package helper

import (
	"os"
	"testing"

	"github.com/loft-sh/devspace/pkg/util/fsutil"

	"gotest.tools/assert"
)

func TestCreateTempDockerfile(t *testing.T) {
	//Create tempDir and go into it
	dir := t.TempDir()

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
	}()

	err = fsutil.WriteToFile([]byte(""), "Exists")
	dockerfilepath, err := CreateTempDockerfile("Exists", []string{"echo"}, []string{""}, nil, "")
	assert.NilError(t, err, "Error when creating a valid temporary Dockerfile")
	dockerfileContent, err := fsutil.ReadFile(dockerfilepath, -1)
	assert.NilError(t, err, "Temporary Dockerfile not created.")
	assert.Equal(t, "\n\nENTRYPOINT [\"echo\"]\n\n\nCMD [\"\"]\n", string(dockerfileContent), "Temporary dockerfile has wrong content")
}
