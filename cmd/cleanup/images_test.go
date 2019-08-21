package cleanup

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"gotest.tools/assert"
)

var logOutput string

type testLogger struct {
	log.DiscardLogger
}

func (t testLogger) Info(args ...interface{}) {
	logOutput = logOutput + "\nInfo " + fmt.Sprint(args...)
}
func (t testLogger) Infof(format string, args ...interface{}) {
	logOutput = logOutput + "\nInfo " + fmt.Sprintf(format, args...)
}

func (t testLogger) Done(args ...interface{}) {
	logOutput = logOutput + "\nDone " + fmt.Sprint(args...)
}
func (t testLogger) Donef(format string, args ...interface{}) {
	logOutput = logOutput + "\nDone " + fmt.Sprintf(format, args...)
}

func (t testLogger) Warn(args ...interface{}) {
	logOutput = logOutput + "\nWarn " + fmt.Sprint(args...)
}
func (t testLogger) Warnf(format string, args ...interface{}) {
	logOutput = logOutput + "\nWarn " + fmt.Sprintf(format, args...)
}

func (t testLogger) StartWait(message string) {
	logOutput = logOutput + "\nWait " + message
}

type RunCleanupImagesTestCase struct {
	name string

	fakeConfig *latest.Config

	answers []string

	expectedOutput string
	expectedPanic  string
}

func TestRunCleanupImages(t *testing.T) {
	testCases := []RunCleanupImagesTestCase{
		RunCleanupImagesTestCase{
			name:          "No devspace config",
			expectedPanic: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		RunCleanupImagesTestCase{
			name:           "No images to delete",
			fakeConfig:     &latest.Config{},
			expectedOutput: "\nDone No images found in config to delete",
		},
		RunCleanupImagesTestCase{
			name: "One image to delete",
			fakeConfig: &latest.Config{
				Images: &map[string]*latest.ImageConfig{
					"imageToDelete": &latest.ImageConfig{
						Image: ptr.String("imageToDelete"),
					},
				},
			},
			expectedOutput: "\nWait Deleting local image imageToDelete\nWait Deleting local dangling images\nDone Successfully cleaned up images",
		},
	}

	for _, testCase := range testCases {
		testRunCleanupImages(t, testCase)
	}

}

func testRunCleanupImages(t *testing.T, testCase RunCleanupImagesTestCase) {
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

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	(&imagesCmd{}).RunCleanupImages(nil, nil)

	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
