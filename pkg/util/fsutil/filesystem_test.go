package fsutil

import (
	"os"
	"testing"

	"gotest.tools/assert"
)

func TestWriteReadFile(t *testing.T) {

	dir := t.TempDir()

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	// 8. Delete temp folder
	defer func() {
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
	}()

	err = WriteToFile([]byte("Some Content"), "someDir/someFile")
	if err != nil {
		t.Fatalf("Error using WriteToFile: %v", err)
	}

	content, err := ReadFile("someDir/someFile", -1)
	if err != nil {
		t.Fatalf("Error using ReadFile without limit: %v", err)
	}
	assert.Equal(t, "Some Content", string(content), "File contains wrong content")

	content, err = ReadFile("someDir/someFile", 4)
	if err != nil {
		t.Fatalf("Error using ReadFile with limit: %v", err)
	}
	assert.Equal(t, "Some", string(content), "File contains wrong content or wrong content returned")

	_, err = ReadFile("", 1)
	assert.Equal(t, true, err != nil, "No error when reading file without filename")
	_, err = ReadFile("someDir", 1)
	assert.Equal(t, true, err != nil, "No error when reading dir like a file")
}

func TestCopy(t *testing.T) {

	dir := t.TempDir()

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	// 8. Delete temp folder
	defer func() {
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
	}()

	err = WriteToFile([]byte("Some Content"), "someDir/someFile")
	if err != nil {
		t.Fatalf("Error using WriteToFile: %v", err)
	}

	_ = Copy("someDir", "copiedDir", false)

	dirInfo, err := os.Stat("copiedDir")
	assert.Equal(t, false, os.IsNotExist(err), "Copy called but no copied dir appeared")
	assert.Equal(t, true, dirInfo.IsDir(), "Copied dir is not a dir")
	dirInfo, err = os.Stat("someDir")
	assert.Equal(t, false, os.IsNotExist(err), "Source dir disappeared")
	assert.Equal(t, true, dirInfo.IsDir(), "Source dir is not a dir anymore")

	content, err := ReadFile("someDir/someFile", -1)
	if err != nil {
		t.Fatalf("Error trying to read source file: %v", err)
	}
	assert.Equal(t, "Some Content", string(content), "Source File corrupted")

	content, err = ReadFile("copiedDir/someFile", -1)
	if err != nil {
		t.Fatalf("Error trying to read copied file: %v", err)
	}
	assert.Equal(t, "Some Content", string(content), "Copied File corrupted")

}
