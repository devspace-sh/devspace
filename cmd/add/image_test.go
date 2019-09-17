package add

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"gotest.tools/assert"
)

type addImageTestCase struct {
	name string

	args       []string
	answers    []string
	fakeConfig *latest.Config
	cmd        *imageCmd

	expectedOutput   string
	expectedErr      string
	expectConfigFile bool
	expectedImages   map[string]*latest.ImageConfig
}

func TestRunAddImage(t *testing.T) {
	testCases := []addImageTestCase{
		addImageTestCase{
			name:        "No devspace config",
			args:        []string{""},
			expectedErr: "Couldn't find a DevSpace configuration. Please run `devspace init`",
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

		assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
	}()

	if testCase.cmd == nil {
		testCase.cmd = &imageCmd{}
	}
	if testCase.cmd.GlobalFlags == nil {
		testCase.cmd.GlobalFlags = &flags.GlobalFlags{}
	}

	err = (testCase.cmd).RunAddImage(nil, testCase.args)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
		return
	}

	config, err := configutil.GetBaseConfig(&configutil.ConfigOptions{})
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, len(testCase.expectedImages), len(config.Images), "Wrong number of images in testCase %s", testCase.name)
	for nameInConfig, image := range config.Images {
		assert.Equal(t, testCase.expectedImages[nameInConfig] == nil, false, "Image %s unexpected in testCase %s", nameInConfig, testCase.name)
		assert.Equal(t, testCase.expectedImages[nameInConfig].Image, image.Image, "Image %s has unexpected name in testCase %s", nameInConfig, testCase.name)
		assert.Equal(t, testCase.expectedImages[nameInConfig].Tag, image.Tag, "Image %s has unexpected tag in testCase %s", nameInConfig, testCase.name)
		assert.Equal(t, testCase.expectedImages[nameInConfig].Context, image.Context, "Image %s has unexpected context in testCase %s", nameInConfig, testCase.name)
		assert.Equal(t, testCase.expectedImages[nameInConfig].Dockerfile, image.Dockerfile, "Image %s has unexpected dockerfile path in testCase %s", nameInConfig, testCase.name)
	}

	err = os.Remove(constants.DefaultConfigPath)
	assert.Equal(t, !os.IsNotExist(err), testCase.expectConfigFile, "Unexpectedly saved or not saved in testCase %s", testCase.name)
}
