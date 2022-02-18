package hash

import (
	"os"
	"testing"

	"github.com/loft-sh/devspace/pkg/util/fsutil"

	"gotest.tools/assert"
)

func TestHashPassword(t *testing.T) {
	hashed, err := Password("password")
	if err != nil {
		t.Fatalf("Error hashing password %s: %v", "password", err)
	}
	assert.Equal(t, "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8", hashed, "Wrong hash returned")
}

func TestHashString(t *testing.T) {
	hashed := String("string")
	assert.Equal(t, "473287f8298dba7163a897908958f7c0eae733e25d2e027992ea2edc9bed2fa8", hashed, "Wrong hash returned")
}

func TestHashDirectory(t *testing.T) {
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

	//Use on empty dir
	_, err = Directory(".")
	if err != nil {
		t.Fatalf("Error creating hash of directory: %v", err)
	}

	//Use on file
	_ = fsutil.WriteToFile([]byte(""), "someFile")
	_, err = Directory("someFile")
	if err != nil {
		t.Fatalf("Error creating hash of file: %v", err)
	}

}

func TestHashDirectoryExcludes(t *testing.T) {
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

	_ = fsutil.WriteToFile([]byte(""), "inludedFile")
	_ = fsutil.WriteToFile([]byte(""), "excludedFile")
	_ = fsutil.WriteToFile([]byte(""), "excludedDir/someFile")
	_, err = DirectoryExcludes(".", []string{"excludedFile", "excludedDir"}, false)
	if err != nil {
		t.Fatalf("Error creating hash of directory: %v", err)
	}

}
