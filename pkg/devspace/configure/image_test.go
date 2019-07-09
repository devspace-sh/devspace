package configure

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"gotest.tools/assert"
)

func TestGetImageConfigFromDockerfile(t *testing.T) {
	testConfig := &latest.Config{}
	_, err := GetImageConfigFromDockerfile(testConfig, "", "", ptr.String("invalid"))
	if err == nil {
		t.Fatalf("No error getting image config from dockerfile with invalid provider.")
	}
}

func TestAddAndRemoveImage(t *testing.T) { //Create tempDir and go into it
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

	configutil.SetFakeConfig(&latest.Config{})
	config := configutil.GetBaseConfig()
	config.Images = nil
	fsutil.WriteToFile([]byte(""), "devspace.yaml")
	AddImage("NewTestImage", "TestImageName", "vTest", "mycontext", "Dockerfile", "docker")
	assert.Equal(t, 1, len(*config.Images), "New image not added: Wrong number of images")
	assert.Equal(t, "TestImageName", *(*config.Images)["NewTestImage"].Image, "New image not correctly added")
	assert.Equal(t, "vTest", *(*config.Images)["NewTestImage"].Tag, "New image not correctly added")

	AddImage("SecoundTestImage", "SecoundImageName", "v2", "mycontext", "Dockerfile", "kaniko")
	assert.Equal(t, 2, len(*config.Images), "New image not added: Wrong number of images")
	assert.Equal(t, "SecoundImageName", *(*config.Images)["SecoundTestImage"].Image, "New image not correctly added")
	assert.Equal(t, "v2", *(*config.Images)["SecoundTestImage"].Tag, "New image not correctly added")

	AddImage("ThirdTestImage", "ThirdImageName", "v3", "mycontext", "Dockerfile", "wrongBuildEngine")
	assert.Equal(t, 3, len(*config.Images), "New image not added: Wrong number of images")
	assert.Equal(t, "ThirdImageName", *(*config.Images)["ThirdTestImage"].Image, "New image not correctly added")
	assert.Equal(t, "v3", *(*config.Images)["ThirdTestImage"].Tag, "New image not correctly added")

	err = RemoveImage(false, []string{})
	assert.Error(t, err, "You have to specify at least one image")

	err = RemoveImage(false, []string{"Doesn'tExist"})
	assert.NilError(t, err, "Error removing non existent image: %v")
	assert.Equal(t, 3, len(*config.Images), "RemoveImage removed an image that wasn't specified")

	err = RemoveImage(false, []string{"SecoundTestImage"})
	assert.NilError(t, err, "Error removing existent image: %v")
	assert.Equal(t, 2, len(*config.Images), "RemoveImage doesn't remove a specified image")
	assert.Equal(t, true, (*config.Images)["SecoundTestImage"] == nil, "RemoveImage removed wrong image")
}
