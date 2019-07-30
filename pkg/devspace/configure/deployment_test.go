package configure

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"gotest.tools/assert"
)

type GetDockerfileComponentDeploymentTestCase struct {
	name string

	answers           []string
	nameParam         string
	imageName         string
	dockerfile        string
	dockerfileContent string
	context           string

	expectedErr            string
	expectedImage          string
	expectedTag            string
	expectedDockerfile     string
	expectedContext        string
	expectedDeploymentName string
	expectedPort           int
}

func TestGetDockerfileComponentDeployment(t *testing.T) {
	testCases := []GetDockerfileComponentDeploymentTestCase{
		GetDockerfileComponentDeploymentTestCase{
			name:          "Empty params, only answers",
			answers:       []string{"someRegistry.com", "someRegistry.com/user/imagename", "yes", "1234"},
			expectedImage: "someRegistry.com/user/imagename",
			expectedPort:  1234,
		},
		GetDockerfileComponentDeploymentTestCase{
			name:               "No answers, only 1 port in dockerfile",
			answers:            []string{},
			imageName:          "someImage",
			dockerfile:         "customDockerFile",
			dockerfileContent:  `EXPOSE 1010`,
			expectedImage:      "someImage",
			expectedDockerfile: "customDockerFile",
			expectedPort:       1010,
		},
		GetDockerfileComponentDeploymentTestCase{
			name:       "2 ports in dockerfile",
			answers:    []string{""},
			imageName:  "someImage",
			dockerfile: "customDockerFile",
			dockerfileContent: `EXPOSE 1011
EXPOSE 1012`,
			expectedImage:      "someImage",
			expectedDockerfile: "customDockerFile",
			expectedPort:       1011,
		},
		GetDockerfileComponentDeploymentTestCase{
			name:        "Invalid port",
			answers:     []string{"someRegistry.com", "someRegistry.com/user/imagename", "yes", "hello"},
			expectedErr: "parsing port: strconv.Atoi: parsing \"hello\": invalid syntax",
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

	for _, testCase := range testCases {
		testConfig := &latest.Config{}
		generated := &generated.Config{}

		if testCase.dockerfile != "" {
			err = fsutil.WriteToFile([]byte(testCase.dockerfileContent), testCase.dockerfile)
		}

		for _, answer := range testCase.answers {
			survey.SetNextAnswer(answer)
		}

		imageConfig, deploymentConfig, err := GetDockerfileComponentDeployment(testConfig, generated, testCase.nameParam, testCase.imageName, testCase.dockerfile, testCase.context)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
			assert.Equal(t, ptr.ReverseString(imageConfig.Image), testCase.expectedImage, "Returned image is unexpected in testCase %s", testCase.name)
			assert.Equal(t, ptr.ReverseString(imageConfig.Tag), testCase.expectedTag, "Returned tag is unexpected in testCase %s", testCase.name)
			assert.Equal(t, ptr.ReverseString(imageConfig.Dockerfile), testCase.expectedDockerfile, "Returned dockerfile is unexpected in testCase %s", testCase.name)
			assert.Equal(t, ptr.ReverseString(imageConfig.Context), testCase.expectedContext, "Returned context is unexpected in testCase %s", testCase.name)
			assert.Equal(t, ptr.ReverseString(deploymentConfig.Name), testCase.expectedDeploymentName, "Returned deployment name is unexpected in testCase %s", testCase.name)
			assert.Equal(t, *(*deploymentConfig.Component.Service.Ports)[0].Port, testCase.expectedPort, "Returned port in deployment is unexpected in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
	}
}
