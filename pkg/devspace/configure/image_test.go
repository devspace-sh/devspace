package configure

import (
	"io/ioutil"
	"os"
	"testing"

	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"gotest.tools/assert"
)

type GetImageConfigFromDockerfileTestCase struct {
	name string

	cloudProvider *string
	answers       []string
	dockerfile    string
	context       string

	expectedErr        string
	expectedImage      string
	expectedTag        string
	expectedDockerfile string
	expectedContext    string
}

func TestGetImageConfigFromDockerfile(t *testing.T) {
	providerConfig, err := cloudconfig.ParseProviderConfig()
	assert.NilError(t, err, "Error getting provider config")
	cloudProviders := ""
	for _, p := range providerConfig.Providers {
		cloudProviders += p.Name + " "
	}

	testCases := []GetImageConfigFromDockerfileTestCase{
		GetImageConfigFromDockerfileTestCase{
			name:          "invalid Cloud provider",
			cloudProvider: ptr.String("invalid"),
			expectedErr:   "Error login into cloud provider: Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: " + cloudProviders,
		},
		GetImageConfigFromDockerfileTestCase{
			name:          "unknown cloud provider from question",
			answers:       []string{"someRegistry.com", "someRegistry.com/user/imagename", "yes"},
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

	for _, testCase := range testCases {
		testConfig := &latest.Config{}

		for _, answer := range testCase.answers {
			survey.SetNextAnswer(answer)
		}

		imageConfig, err := GetImageConfigFromDockerfile(testConfig, testCase.dockerfile, testCase.context, testCase.cloudProvider)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
			assert.Equal(t, ptr.ReverseString(imageConfig.Image), testCase.expectedImage, "Returned image is unexpected in testCase %s", testCase.name)
			assert.Equal(t, ptr.ReverseString(imageConfig.Tag), testCase.expectedTag, "Returned tag is unexpected in testCase %s", testCase.name)
			assert.Equal(t, ptr.ReverseString(imageConfig.Dockerfile), testCase.expectedDockerfile, "Returned dockerfile is unexpected in testCase %s", testCase.name)
			assert.Equal(t, ptr.ReverseString(imageConfig.Context), testCase.expectedContext, "Returned context is unexpected in testCase %s", testCase.name)
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
			name:        "No pull secrets",
			answers:     []string{"no"},
			expectedNil: true,
		},
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
			assert.Equal(t, ptr.ReverseString(imageConfig.Image), testCase.expectedImage, "Returned image is unexpected in testCase %s", testCase.name)
			assert.Equal(t, ptr.ReverseString(imageConfig.Tag), testCase.expectedTag, "Returned tag is unexpected in testCase %s", testCase.name)
			assert.Equal(t, ptr.ReverseString(imageConfig.Dockerfile), testCase.expectedDockerfile, "Returned dockerfile is unexpected in testCase %s", testCase.name)
			assert.Equal(t, ptr.ReverseString(imageConfig.Context), testCase.expectedContext, "Returned context is unexpected in testCase %s", testCase.name)
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
