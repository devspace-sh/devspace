package remove

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"gotest.tools/assert"
)

type removeImageTestCase struct {
	name string

	fakeConfig *latest.Config

	args      []string
	answers   []string
	removeAll bool

	expectedOutput   string
	expectedPanic    string
	expectConfigFile bool
}

func TestRunRemoveImage(t *testing.T) {
	testCases := []removeImageTestCase{
		removeImageTestCase{
			name:          "No devspace config",
			args:          []string{""},
			expectedPanic: "Couldn't find any devspace configuration. Please run `devspace init`",
		},
		removeImageTestCase{
			name:          "No image specified",
			fakeConfig:    &latest.Config{},
			expectedPanic: "You have to specify at least one image",
		},
		removeImageTestCase{
			name:             "Remove all zero images",
			fakeConfig:       &latest.Config{},
			removeAll:        true,
			expectedOutput:   "\nDone Successfully removed all images",
			expectConfigFile: true,
		},
		removeImageTestCase{
			name:             "Remove not existent image",
			fakeConfig:       &latest.Config{},
			args:             []string{"Doesn'tExist"},
			expectedOutput:   "\nDone Successfully removed image Doesn'tExist",
			expectConfigFile: true,
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testRunRemoveImage(t, testCase)
	}
}

func testRunRemoveImage(t *testing.T, testCase removeImageTestCase) {
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

	isImagesNil := testCase.fakeConfig == nil || testCase.fakeConfig.Images == nil
	configutil.SetFakeConfig(testCase.fakeConfig)
	if isImagesNil && testCase.fakeConfig != nil {
		testCase.fakeConfig.Images = nil
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
		RemoveAll: testCase.removeAll,
	}).RunRemoveImage(nil, testCase.args)

	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)

	err = os.Remove(constants.DefaultConfigPath)
	assert.Equal(t, !os.IsNotExist(err), testCase.expectConfigFile, "Unexpectedly saved or not saved in testCase %s", testCase.name)
}
