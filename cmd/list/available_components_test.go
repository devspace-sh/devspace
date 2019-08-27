package list

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/generator"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
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

	expectedOutput string
	expectedPanic  string
}

func TestListAvailableComponents(t *testing.T) {
	testCases := []listAvailableComponentsTestCase{
		listAvailableComponentsTestCase{
			name: "List components",
			expectedOutput: "\n" + ansi.Color(" Name  ", "green+b") + "    " + ansi.Color(" Description  ", "green+b") + "                                                                             " + `
 mariadb    MariaDB is a community-developed fork of MySQL intended to remain free under the GNU GPL  
 mongodb    MongoDB document databases provide high availability and easy scalability                 
 mysql      MySQL is a widely used, open-source relational database management system (RDBMS)         
 postgres   The PostgreSQL object-relational database system provides reliability and data integrity  
 redis      Redis is an open source key-value store that functions as a data structure server         

`,
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
	logOutput = ""

	defer func() {
		rec := recover()
		if testCase.expectedPanic == "" {
			if rec != nil {
				t.Fatalf("Unexpected panic in testCase %s. Message: %s. Stack: %s", testCase.name, rec, string(debug.Stack()))
			}
		} else {
			if rec == nil {
				t.Fatalf("Unexpected no panic in testCase %s", testCase.name)
			} else {
				assert.Equal(t, rec, testCase.expectedPanic, "Wrong panic message in testCase %s. Stack: %s", testCase.name, string(debug.Stack()))
			}
		}
		assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
	}()

	(&availableComponentsCmd{}).RunListAvailableComponents(nil, []string{})

	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
}
