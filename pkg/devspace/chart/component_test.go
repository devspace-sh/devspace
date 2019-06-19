package chart

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/generator"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	homedir "github.com/mitchellh/go-homedir"

	"gotest.tools/assert"
)

func TestListAvailableComponents(t *testing.T) {
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

	//Backup components
	homedir, err := homedir.Dir()
	assert.NilError(t, err, "Error getting homedir")
	err = os.Rename(filepath.Join(homedir, generator.ComponentsRepoPath, "components"), filepath.Join(dir, "backup"))
	assert.NilError(t, err, "Error making a backup for the components")
	defer fsutil.Copy(filepath.Join(dir, "backup"), filepath.Join(homedir, generator.ComponentsRepoPath, "components"), true)

	//Delete components
	err = os.RemoveAll(filepath.Join(homedir, generator.ComponentsRepoPath, "components"))
	assert.NilError(t, err, "Error removing the components")
	err = os.MkdirAll(filepath.Join(homedir, generator.ComponentsRepoPath, "components"), 0755)
	assert.NilError(t, err, "Error removing the components")

	availableComponents, err := ListAvailableComponents()
	assert.NilError(t, err, "Error listing available components")
	assert.Equal(t, 0, len(availableComponents), "Unexpected number of available components")
}
