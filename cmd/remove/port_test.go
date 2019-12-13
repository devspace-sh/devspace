package remove

/*
import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"gotest.tools/assert"
)

type removePortTestCase struct {
	name string

	fakeConfig *latest.Config

	args          []string
	answers       []string
	removeAll     bool
	labelSelector string

	expectedErr      string
	expectConfigFile bool
}

func TestRunRemovePort(t *testing.T) {
	testCases := []removePortTestCase{
		removePortTestCase{
			name:        "No port specified",
			fakeConfig:  &latest.Config{},
			expectedErr: "You have to specify at least one of the supported flags",
		},
		removePortTestCase{
			name:       "Remove all zero ports",
			fakeConfig: &latest.Config{},
			removeAll:  true,
		},
	}

	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testRunRemovePort(t, testCase)
	}
}

func testRunRemovePort(t *testing.T, testCase removePortTestCase) {
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

	isPortsNil := testCase.fakeConfig == nil || testCase.fakeConfig.Images == nil
	loader.SetFakeConfig(testCase.fakeConfig)
	if isPortsNil && testCase.fakeConfig != nil {
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
	}()

	err = (&portCmd{
		RemoveAll:     testCase.removeAll,
		LabelSelector: testCase.labelSelector,
		GlobalFlags:   &flags.GlobalFlags{},
	}).RunRemovePort(nil, testCase.args)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}

	err = os.Remove(constants.DefaultConfigPath)
	assert.Equal(t, !os.IsNotExist(err), testCase.expectConfigFile, "Unexpectedly saved or not saved in testCase %s", testCase.name)
}*/
