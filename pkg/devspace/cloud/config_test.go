package cloud

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"

	homedir "github.com/mitchellh/go-homedir"
	"gotest.tools/assert"
)

func TestLoadCloudConfig(t *testing.T) {
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
	config.DevSpaceCloudConfigPath, err = filepath.Rel(homedir, dir)
	assert.NilError(t, err, "Error getting relative path from homedir to current dir")
	config.DevSpaceCloudConfigPath = filepath.Join(config.DevSpaceCloudConfigPath, "DSCloudConfig")

	loadedConfigOnce = sync.Once{}
	providerConfig, err := LoadCloudConfig()
	assert.NilError(t, err, "Error loading Config from non-existent path.")
	assert.Equal(t, providerConfig[config.DevSpaceCloudProviderName], DevSpaceCloudProviderConfig, "ProviderConfig is not set to default")

	err = fsutil.WriteToFile([]byte(""), "DSCloudConfig")
	assert.NilError(t, err, "Error creating empty file")
	loadedConfigOnce = sync.Once{}
	providerConfig, err = LoadCloudConfig()
	assert.NilError(t, err, "Error loading Config from existing path without content.")
	assert.Equal(t, providerConfig[config.DevSpaceCloudProviderName], DevSpaceCloudProviderConfig, "ProviderConfig is not set to default")

	err = fsutil.WriteToFile([]byte(`app.devspace.cloud:
  token: someToken`), "DSCloudConfig")
	assert.NilError(t, err, "Error creating empty file")
	loadedConfigOnce = sync.Once{}
	providerConfig, err = LoadCloudConfig()
	assert.NilError(t, err, "Error loading Config from existing path without content.")
	assert.Equal(t, providerConfig[config.DevSpaceCloudProviderName].Host, DevSpaceCloudProviderConfig.Host, "ProviderConfig's host is not set to default")
	assert.Equal(t, providerConfig[config.DevSpaceCloudProviderName].Token, "someToken", "ProviderConfighas wrong token")
}

func TestSaveConfig(t *testing.T) {
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
	config.DevSpaceCloudConfigPath, err = filepath.Rel(homedir, dir)
	assert.NilError(t, err, "Error getting relative path from homedir to current dir")
	config.DevSpaceCloudConfigPath = filepath.Join(config.DevSpaceCloudConfigPath, "DSCloudConfig")

	providerConfig := ProviderConfig{
		"someProvider": &Provider{
			Name:  "Isn't used at all",
			Host:  "someHost",
			Key:   "someKey",
			Token: "someToken",
			ClusterKey: map[int]string{
				1: "someClusterKey",
			},
		},
	}

	err = SaveCloudConfig(providerConfig)
	assert.NilError(t, err, "Error saving config")

	configContent, err := fsutil.ReadFile("DSCloudConfig", -1)
	assert.NilError(t, err, "Error reading config. Probably because it wasn't saved correctly")
	assert.Equal(t, string(configContent), `someProvider:
  host: someHost
  key: someKey
  token: someToken
  clusterKeys:
    1: someClusterKey
`, "Saved config has wrong content")

}

func TestSave(t *testing.T) {
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
	config.DevSpaceCloudConfigPath, err = filepath.Rel(homedir, dir)
	assert.NilError(t, err, "Error getting relative path from homedir to current dir")
	config.DevSpaceCloudConfigPath = filepath.Join(config.DevSpaceCloudConfigPath, "DSCloudConfig")
	err = fsutil.WriteToFile([]byte(""), "DSCloudConfig")

	provider := &Provider{
		Name:  "someProvider",
		Host:  "someHost",
		Key:   "someKey",
		Token: "someToken",
		ClusterKey: map[int]string{
			1: "someClusterKey",
		},
	}

	err = provider.Save()
	assert.NilError(t, err, "Error saving provider")

	configContent, err := fsutil.ReadFile("DSCloudConfig", -1)
	assert.NilError(t, err, "Error reading config. Probably because it wasn't saved correctly")
	assert.Equal(t, string(configContent), `app.devspace.cloud:
  token: someToken
someProvider:
  host: someHost
  key: someKey
  token: someToken
  clusterKeys:
    1: someClusterKey
`, "Saved config has wrong content")
}
