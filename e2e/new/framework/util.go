package framework

import (
	"github.com/otiai10/copy"
	"io/ioutil"
	"os"
	"sync"
)

func InterruptChan() (chan error, func()) {
	once := sync.Once{}
	c := make(chan error)
	return c, func() {
		once.Do(func() {
			close(c)
		})
	}
}

func CleanupTempDir(initialDir, tempDir string) {
	err := os.RemoveAll(tempDir)
	ExpectNoError(err)

	err = os.Chdir(initialDir)
	ExpectNoError(err)
}

func CopyToTempDir(relativePath string) (string, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	err = copy.Copy(relativePath, dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	err = os.Chdir(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	return dir, nil
}

func ChangeToTempDir() (string, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	err = os.Chdir(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	return dir, nil
}
