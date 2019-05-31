package generator

import (
	"io/ioutil"
	"os"
	"testing"
	
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	
	"gotest.tools/assert"
)

func TestUpdate(t *testing.T){
	//Create TmpFolder
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

	// Cleanup temp folder
	defer os.Chdir(wdBackup)
	defer os.RemoveAll(dir)

	chartGenerator, err := NewChartGenerator("")
	if err != nil {
		t.Fatalf("Error creating new ChartGenerator: %v", err)
	}
	err = chartGenerator.Update(false)
	assert.Equal(t, true, err != nil, "No error when updating without devspace.yaml and no force")

	//The method will try to create this folder and fail
	chartGenerator.LocalPath = "*//"
	err = chartGenerator.Update(false)
	assert.Equal(t, true, err != nil, "No error when using a corrupted local path")

	err = fsutil.WriteToFile([]byte(""), "templates/someFileThatNeedsToBeCleaned")
	if err != nil {
		t.Fatalf("Error writin a file: %v", err)
	}
	chartGenerator.LocalPath = dir
	err = chartGenerator.Update(true)
	if err != nil {
		t.Fatalf("Error calling Update with force: %v", err)
	}
}
