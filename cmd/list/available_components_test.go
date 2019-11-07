package list

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/generator"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	homedir "github.com/mitchellh/go-homedir"

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

func (t testLogger) Fail(args ...interface{}) {
	logOutput = logOutput + "\nFail " + fmt.Sprint(args...)
}
func (t testLogger) Failf(format string, args ...interface{}) {
	logOutput = logOutput + "\nFail " + fmt.Sprintf(format, args...)
}

func (t testLogger) Warn(args ...interface{}) {
	logOutput = logOutput + "\nWarn " + fmt.Sprint(args...)
}
func (t testLogger) Warnf(format string, args ...interface{}) {
	logOutput = logOutput + "\nWarn " + fmt.Sprintf(format, args...)
}

func (t testLogger) StartWait(msg string) {
	logOutput = logOutput + "\nWait " + fmt.Sprint(msg)
}

func (t testLogger) Write(msg []byte) (int, error) {
	logOutput = logOutput + string(msg)
	return len(msg), nil
}

type listAvailableComponentsTestCase struct {
	name string

	expectedHeader []string
	expectedValues [][]string
	expectedErr    string
}

func TestListAvailableComponents(t *testing.T) {
	testCases := []listAvailableComponentsTestCase{
		listAvailableComponentsTestCase{
			name:           "List components",
			expectedHeader: []string{"Name", "Description"},
			expectedValues: [][]string{
				[]string{"mariadb", "MariaDB is a community-developed fork of MySQL intended to remain free under the GNU GPL"},
				[]string{"mongodb", "MongoDB document databases provide high availability and easy scalability"},
				[]string{"mysql", "MySQL is a widely used, open-source relational database management system (RDBMS)"},
				[]string{"postgres", "The PostgreSQL object-relational database system provides reliability and data integrity"},
				[]string{"redis", "Redis is an open source key-value store that functions as a data structure server"},
			},
		},
	}

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

	homedir, err := homedir.Dir()
	assert.NilError(t, err, "Error getting homedir")
	componentDirBackup := filepath.Join(dir, "backup")
	err = fsutil.Copy(filepath.Join(homedir, generator.ComponentsRepoPath), componentDirBackup, false)
	assert.NilError(t, err, "Error creating a backup for the components")

	defer func() {
		err = os.RemoveAll(filepath.Join(homedir, generator.ComponentsRepoPath))
		assert.NilError(t, err, "Error removing component dir")
		err = fsutil.Copy(componentDirBackup, filepath.Join(homedir, generator.ComponentsRepoPath), false)
		assert.NilError(t, err, "Error restoring components")

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

	log.SetInstance(&testLogger{})

	for _, testCase := range testCases {
		testListAvailableComponents(t, testCase)
	}
}

func testListAvailableComponents(t *testing.T, testCase listAvailableComponentsTestCase) {
	log.SetFakePrintTable(func(s log.Logger, header []string, values [][]string) {
		assert.Equal(t, reflect.DeepEqual(header, testCase.expectedHeader), true, "Unexpected header in testCase %s. Expected:%v\nActual:%v", testCase.name, testCase.expectedHeader, header)
		assert.Equal(t, reflect.DeepEqual(values, testCase.expectedValues), true, "Unexpected values in testCase %s. Expected:%v\nActual:%v", testCase.name, testCase.expectedValues, values)
		return
	})

	err := (&availableComponentsCmd{}).RunListAvailableComponents(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}
