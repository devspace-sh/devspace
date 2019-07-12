package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	homedir "github.com/mitchellh/go-homedir"

	"gotest.tools/assert"
)

func TestReadWriteCloudsConfig(t *testing.T) {
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

	homedir, err := homedir.Dir()
	assert.NilError(t, err, "Error getting homedir")
	DevSpaceCloudConfigPath, err = filepath.Rel(homedir, dir)
	assert.NilError(t, err, "Error getting relative path from homedir to current dir")
	DevSpaceCloudConfigPath = filepath.Join(DevSpaceCloudConfigPath, "cloudConfig.yaml")

	err = SaveCloudsConfig([]byte("Hello World"))
	assert.NilError(t, err, "Error saving cloudsConfig")
	readData, err := ReadCloudsConfig()
	assert.NilError(t, err, "Error reading cloudsConfig")
	assert.Equal(t, "Hello World", string(readData), "CloudConfig changed during save/read process")
}
