package envutil

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/util/randutil"
	
	"gotest.tools/assert"
)

func TestAddToPath(t *testing.T) {
	pathBackup := os.Getenv("PATH")
	defer func(){
		SetEnvVar("PATH", pathBackup)
		assert.Equal(t, pathBackup, os.Getenv("PATH"), "This is bad. Path couldn't be reset to previous state: " + pathBackup)
	}()
	t.Log("Before:")
	t.Log(pathBackup)
	
	pathSeparator := ":"

	if runtime.GOOS == "windows" {
		pathSeparator = ";"
	}
	paths := strings.Split(pathBackup, pathSeparator)

	newPath, err := randutil.GenerateRandomString(20)
	if err != nil {
		t.Fatalf("Error choosing a new path: %v", err)
	}
	for includes(paths, newPath) {
		newPath, err = randutil.GenerateRandomString(20)
		if err != nil {
			t.Fatalf("Error choosing a new path: %v", err)
		}
	}

	err = AddToPath(newPath)
	if err != nil {
		t.Fatalf("Error adding new path: %v", err)
	}

	pathAfterAdding := os.Getenv("PATH")
	pathsAfterAdding := strings.Split(pathAfterAdding, pathSeparator)
	t.Log("After:")
	t.Log(pathAfterAdding)
	assert.Equal(t, true, includes(pathsAfterAdding, newPath), "The new path wasn't successfully added")
}

func includes(arr []string, needle string) bool{
	for _, elem := range arr {
		if elem == needle {
			return true
		}
	}
	return false
}
