package v2

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"k8s.io/helm/pkg/helm/environment"

	"gotest.tools/assert"
)

type locateChartPathTestCase struct {
	name string

	settedDirs  []string
	settedFiles []string

	settings  *environment.EnvSettings
	repoURL   string
	username  string
	password  string
	nameParam string
	version   string
	verify    bool
	keyring   string
	certFile  string
	keyFile   string
	caFile    string

	expectedErr                               string
	expectedReturnedString                    string
	expectReturnedStringToBeAbsolutePathInDir bool
}

func TestLocateChartPathDependencies(t *testing.T) {
	testCases := []locateChartPathTestCase{
		locateChartPathTestCase{
			name:                   "name exists no verification",
			settedFiles:            []string{"someFile.abc"},
			nameParam:              "someFile.abc",
			expectedReturnedString: "someFile.abc",
			expectReturnedStringToBeAbsolutePathInDir: true,
		},
	}

	for _, testCase := range testCases {
		dir, err := ioutil.TempDir("", "test")
		if err != nil {
			t.Fatalf("Error creating temporary directory: %v", err)
		}
		dir, err = filepath.EvalSymlinks(dir)
		assert.NilError(t, err, "Error extending expected returned string in testCase %s", testCase.name)

		wdBackup, err := os.Getwd()
		if err != nil {
			t.Fatalf("Error getting current working directory: %v", err)
		}
		err = os.Chdir(dir)
		if err != nil {
			t.Fatalf("Error changing working directory: %v", err)
		}

		fileInfo, err := os.Lstat(".")
		assert.NilError(t, err, "Error getting local file mode in testCase %s", testCase.name)
		for _, dir := range testCase.settedDirs {
			err = os.Mkdir(dir, fileInfo.Mode())
			assert.NilError(t, err, "Error creating dir in testCase %s", testCase.name)
		}
		for _, file := range testCase.settedFiles {
			fsutil.WriteToFile([]byte(""), file)
			assert.NilError(t, err, "Error creating file in testCase %s", testCase.name)
		}

		returnedString, err := locateChartPath(testCase.settings, testCase.repoURL, testCase.username, testCase.password, testCase.nameParam, testCase.version, testCase.verify, testCase.keyring, testCase.certFile, testCase.keyFile, testCase.caFile)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		if testCase.expectReturnedStringToBeAbsolutePathInDir {
			testCase.expectedReturnedString = filepath.Join(dir, testCase.expectedReturnedString)
		}
		assert.Equal(t, returnedString, testCase.expectedReturnedString, "Wrong string returned in testCase %s", testCase.name)

		//Delete temp folder
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}
}
