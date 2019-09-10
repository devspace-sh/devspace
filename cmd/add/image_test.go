package add

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"gotest.tools/assert"
)

type addImageTestCase struct {
	name string

	args                []string
	answers             []string
	fakeConfig          *latest.Config
	imageName           string
	imageTag            string
	imageContextPath    string
	imageDockerfilePath string
	imageBuildTool    string

	expectedOutput   string
	expectedPanic    string
	expectConfigFile bool
	expectedImages   map[string]*latest.ImageConfig
}

func TestRunAddImage(t *testing.T) {
	testCases := []addImageTestCase{
		addImageTestCase{
			name:          "No devspace config",
			args:          []string{""},
			expectedPanic: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		addImageTestCase{
			name:             "Add one empty image",
			args:             []string{""},
			fakeConfig:       &latest.Config{},
			expectedOutput:   "\nDone Successfully added image ",
			expectConfigFile: true,
			expectedImages: map[string]*latest.ImageConfig{
				"": &latest.ImageConfig{},
			},
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testRunAddImage(t, testCase)
	}
}

func testRunAddImage(t *testing.T, testCase addImageTestCase) {
	logOutput = ""

	dir, err := ioutil.TempDir("", "test")
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

	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	isDeploymentsNil := testCase.fakeConfig == nil || testCase.fakeConfig.Deployments == nil
	configutil.SetFakeConfig(testCase.fakeConfig)
	if isDeploymentsNil && testCase.fakeConfig != nil {
		testCase.fakeConfig.Deployments = nil
	}

	defer func() {
		//Delete temp folder
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}

		rec := recover()
		if testCase.expectedPanic == "" {
			if rec != nil {
				t.Fatalf("Unexpected panic in testCase %s. Message: %s", testCase.name, rec)
			}
		} else {
			if rec == nil {
				t.Fatalf("Unexpected no panic in testCase %s", testCase.name)
			} else {
				assert.Equal(t, rec, testCase.expectedPanic, "Wrong panic message in testCase %s", testCase.name)
			}
		}
		assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
	}()

	(&imageCmd{
		Name:           testCase.imageName,
		Tag:            testCase.imageTag,
		ContextPath:    testCase.imageContextPath,
		DockerfilePath: testCase.imageDockerfilePath,
		BuildTool:    testCase.imageBuildTool,
	}).RunAddImage(nil, testCase.args)

	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)

	config := configutil.GetBaseConfig()

	assert.Equal(t, len(testCase.expectedImages), len(*config.Images), "Wrong number of images in testCase %s", testCase.name)
	for nameInConfig, image := range *config.Images {
		assert.Equal(t, testCase.expectedImages[nameInConfig] == nil, false, "Image %s unexpected in testCase %s", nameInConfig, testCase.name)
		assert.Equal(t, ptr.ReverseString(testCase.expectedImages[nameInConfig].Image), ptr.ReverseString(image.Image), "Image %s has unexpected name in testCase %s", nameInConfig, testCase.name)
		assert.Equal(t, ptr.ReverseString(testCase.expectedImages[nameInConfig].Tag), ptr.ReverseString(image.Tag), "Image %s has unexpected tag in testCase %s", nameInConfig, testCase.name)
		assert.Equal(t, ptr.ReverseString(testCase.expectedImages[nameInConfig].Context), ptr.ReverseString(image.Context), "Image %s has unexpected context in testCase %s", nameInConfig, testCase.name)
		assert.Equal(t, ptr.ReverseString(testCase.expectedImages[nameInConfig].Dockerfile), ptr.ReverseString(image.Dockerfile), "Image %s has unexpected dockerfile path in testCase %s", nameInConfig, testCase.name)
	}

	err = os.Remove(constants.DefaultConfigPath)
	assert.Equal(t, !os.IsNotExist(err), testCase.expectConfigFile, "Unexpectedly saved or not saved in testCase %s", testCase.name)
}
