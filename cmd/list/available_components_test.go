package list

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/generator"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	homedir "github.com/mitchellh/go-homedir"

	"gotest.tools/assert"
)

type listAvailableComponentsTestCase struct {
	name string

	expectedHeader []string
	expectedValues [][]string
	expectedErr    string
}

func TestListAvailableComponents(t *testing.T, f factory.Factory) {
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

	log.SetInstance(log.Discard)

	for _, testCase := range testCases {
		testListAvailableComponents(f, t, testCase)
	}
}

func testListAvailableComponents(f factory.Factory, t *testing.T, testCase listAvailableComponentsTestCase) {
	log.SetFakePrintTable(func(s log.Logger, header []string, values [][]string) {
		assert.Equal(t, reflect.DeepEqual(header, testCase.expectedHeader), true, "Unexpected header in testCase %s. Expected:%v\nActual:%v", testCase.name, testCase.expectedHeader, header)
		assert.Equal(t, reflect.DeepEqual(values, testCase.expectedValues), true, "Unexpected values in testCase %s. Expected:%v\nActual:%v", testCase.name, testCase.expectedValues, values)
		return
	})

	err := (&availableComponentsCmd{}).RunListAvailableComponents(f, nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}
