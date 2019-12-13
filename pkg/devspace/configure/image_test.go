package configure

/*
import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	gohomedir "github.com/mitchellh/go-homedir"

	"gotest.tools/assert"
)

type GetImageConfigFromDockerfileTestCase struct {
	name string

	cloudProvider *string
	providersYaml string
	answers       []string
	dockerfile    string
	context       string

	expectedErr        string
	expectedImage      string
	expectedTag        string
	expectedDockerfile string
	expectedContext    string
}

const defaultProvidersYaml = `version: v1beta1
providers:
- clusterKeys:
    2: "5678901"
    5: "5678901"
- name: app.devspace.cloud
  host: https://app.devspace.cloud
  key: "5678901"
  token: eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImtpZCI6IjNMQUI6U1NTVDpTUElVOlBUUkw6VlVSUTpaV1VROkJJVEY6SUlBNjpFQjNUOjZHWkY6SEVDUDpLRVNVIn0.eyJzdWIiOiIxNTIyNzg0IiwiYWRtaW4iOmZhbHNlLCJpYXQiOjE1NTIzMDY2MDgsImh0dHBzOi8vaGFzdXJhLmlvL2p3dC9jbGFpbXMiOnsieC1oYXN1cmEtdXNlci1pZCI6IjEyIiwieC1oYXN1cmEtZGVmYXVsdC1yb2xlIjoidXNlciIsIngtaGFzdXJhLWFsbG93ZWQtcm9sZXMiOlsidXNlciJdfX0.FbTdo7HLg4C9le1nKhGxiP-g5l9a9EklQUGIrhPIOT8ft1mhecXYcfZrfmquMZY5QHFpoW5wKdpZS95WiKD2fZGqhYn8NgR400Hc-Gisbg1zjH0SnbJZiSIsKw9TPuoPH5GDZ03NWcd0--ifVjUH7oNh1AqFxpsVdjSvV5PhS0lrvjdrqvafNBnjcHylVVTLp-z7IyFuBPh6-rO9LzTfuy73pzLpWGgn9CO26oy7V6shNNn07khkjRr91I7KCKnhMtuCHbzrF1Ma4WHu63TsTmvi-LS2veRmz3edMwWAIdM6BB8HwDy8vtpgM1Kr4nOlLFNa2hZw8FUjC6Cog1D_2A`

func TestGetImageConfigFromDockerfile(t *testing.T) {
	t.Skip("Dependent on docker installation")
	testCases := []GetImageConfigFromDockerfileTestCase{
		GetImageConfigFromDockerfileTestCase{
			name:          "unknown cloud provider from question",
			answers:       []string{"someRegistry.com", "someRegistry.com/user/imagename", "yes"},
			expectedImage: "someRegistry.com/user/imagename",
		},
		GetImageConfigFromDockerfileTestCase{
			name:          "prompt username",
			answers:       []string{"hub.docker.com", "some/image", "yes"},
			expectedImage: "someRegistry.com/user/imagename",
		},
		GetImageConfigFromDockerfileTestCase{
			name:          "use hub.docker.com",
			answers:       []string{"hub.docker.com", "some/image", "yes"},
			expectedImage: "some/image",
		},
		GetImageConfigFromDockerfileTestCase{
			name:               "use gcr",
			answers:            []string{"gcr.io", "other/image", "yes"},
			dockerfile:         "SomeOtherDockerfile",
			context:            "SomeContext",
			expectedImage:      "other/image",
			expectedDockerfile: "SomeOtherDockerfile",
			expectedContext:    "SomeContext",
		},
	}

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

	homedir, err := gohomedir.Dir()
	assert.NilError(t, err, "Error finding out home directory")
	backupProviderPath := cloudconfig.DevSpaceProvidersConfigPath
	cloudconfig.DevSpaceProvidersConfigPath, err = filepath.Rel(homedir, filepath.Join(dir, "providers.yaml"))
	assert.NilError(t, err, "Error getting relative path of temp dir")

	// Delete temp folder after test
	defer func() {
		cloudconfig.DevSpaceProvidersConfigPath = backupProviderPath

		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	for _, testCase := range testCases {
		testConfig := &latest.Config{}

		for _, answer := range testCase.answers {
			survey.SetNextAnswer(answer)
		}

		if testCase.providersYaml == "" {
			testCase.providersYaml = defaultProvidersYaml
		}
		fsutil.WriteToFile([]byte(testCase.providersYaml), filepath.Join(dir, "providers.yaml"))

		imageConfig, err := GetImageConfigFromDockerfile(testConfig, testCase.dockerfile, testCase.context, "devspace", log.GetInstance())
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
			assert.Equal(t, imageConfig.Image, testCase.expectedImage, "Returned image is unexpected in testCase %s", testCase.name)
			assert.Equal(t, imageConfig.Tag, testCase.expectedTag, "Returned tag is unexpected in testCase %s", testCase.name)
			assert.Equal(t, imageConfig.Dockerfile, testCase.expectedDockerfile, "Returned dockerfile is unexpected in testCase %s", testCase.name)
			assert.Equal(t, imageConfig.Context, testCase.expectedContext, "Returned context is unexpected in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
	}
}

type GetImageConfigFromImageNameTestCase struct {
	name string

	answers    []string
	imageName  string
	dockerfile string
	context    string

	expectedNil           bool
	expectedImage         string
	expectedTag           string
	expectedDockerfile    string
	expectedContext       string
	expectedBuildDisabled bool
}

func TestGetImageConfigFromImageName(t *testing.T) {
	testCases := []GetImageConfigFromImageNameTestCase{
		GetImageConfigFromImageNameTestCase{
			name:                  "Empty params with pull secrets",
			answers:               []string{"yes"},
			expectedTag:           "latest",
			expectedBuildDisabled: true,
		},
		GetImageConfigFromImageNameTestCase{
			name:               "All params with pull secrets",
			answers:            []string{"yes"},
			imageName:          "Many:Splitted:Tokens",
			dockerfile:         "customDockerfile",
			context:            "customContext",
			expectedImage:      "Many",
			expectedTag:        "Splitted",
			expectedDockerfile: "customDockerfile",
			expectedContext:    "customContext",
		},
	}

	for _, testCase := range testCases {
		for _, answer := range testCase.answers {
			survey.SetNextAnswer(answer)
		}

		imageConfig := GetImageConfigFromImageName(testCase.imageName, testCase.dockerfile, testCase.context)

		if !testCase.expectedNil {
			if imageConfig == nil {
				t.Fatalf("Nil returned in testCase %s", testCase.name)
			}
			assert.Equal(t, imageConfig.Image, testCase.expectedImage, "Returned image is unexpected in testCase %s", testCase.name)
			assert.Equal(t, imageConfig.Tag, testCase.expectedTag, "Returned tag is unexpected in testCase %s", testCase.name)
			assert.Equal(t, imageConfig.Dockerfile, testCase.expectedDockerfile, "Returned dockerfile is unexpected in testCase %s", testCase.name)
			assert.Equal(t, imageConfig.Context, testCase.expectedContext, "Returned context is unexpected in testCase %s", testCase.name)
			assert.Equal(t, imageConfig.Build != nil && ptr.ReverseBool(imageConfig.Build.Disabled), testCase.expectedBuildDisabled, "Returned build status is unexpected in testCase %s", testCase.name)
		} else {
			if imageConfig != nil {
				t.Fatalf("Not nil returned in testCase %s", testCase.name)
			}
		}
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

	loader.SetFakeConfig(&latest.Config{})
	config, err := loader.GetBaseConfig(nil)
	if err != nil {
		log.Fatal(err)
	}

	config.Images = nil
	fsutil.WriteToFile([]byte(""), "devspace.yaml")
	AddImage(config, "NewTestImage", "TestImageName", "vTest", "mycontext", "Dockerfile", "docker")
	assert.Equal(t, 1, len(config.Images), "New image not added: Wrong number of images")
	assert.Equal(t, "TestImageName", config.Images["NewTestImage"].Image, "New image not correctly added")
	assert.Equal(t, "vTest", config.Images["NewTestImage"].Tag, "New image not correctly added")

	AddImage(config, "SecoundTestImage", "SecoundImageName", "v2", "mycontext", "Dockerfile", "kaniko")
	assert.Equal(t, 2, len(config.Images), "New image not added: Wrong number of images")
	assert.Equal(t, "SecoundImageName", config.Images["SecoundTestImage"].Image, "New image not correctly added")
	assert.Equal(t, "v2", config.Images["SecoundTestImage"].Tag, "New image not correctly added")

	AddImage(config, "ThirdTestImage", "ThirdImageName", "v3", "mycontext", "Dockerfile", "wrongBuildEngine")
	assert.Equal(t, 3, len(config.Images), "New image not added: Wrong number of images")
	assert.Equal(t, "ThirdImageName", config.Images["ThirdTestImage"].Image, "New image not correctly added")
	assert.Equal(t, "v3", config.Images["ThirdTestImage"].Tag, "New image not correctly added")

	err = RemoveImage(config, false, []string{})
	assert.Error(t, err, "You have to specify at least one image")

	err = RemoveImage(config, false, []string{"Doesn'tExist"})
	assert.NilError(t, err, "Error removing non existent image: %v")
	assert.Equal(t, 3, len(config.Images), "RemoveImage removed an image that wasn't specified")

	err = RemoveImage(config, false, []string{"SecoundTestImage"})
	assert.NilError(t, err, "Error removing existent image: %v")
	assert.Equal(t, 2, len(config.Images), "RemoveImage doesn't remove a specified image")
	assert.Equal(t, true, config.Images["SecoundTestImage"] == nil, "RemoveImage removed wrong image")
}*/
