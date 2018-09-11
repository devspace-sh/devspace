package sync

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TODO: CopyToContainer test
func TestCopyToContainerTestable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on windows")
	}

	remote, local := initTestDirs(t)
	defer os.RemoveAll(remote)
	defer os.RemoveAll(local)
	containerPath := "/testDir"

	syncClient := createTestSyncClient(local, remote)

	syncClient.errorChan = make(chan error)

	excludePaths := []string{}

	// Write local files
	ioutil.WriteFile(path.Join(local, "testFile1"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFile2"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "ignoredFile"), []byte(fileContents), 0666)
	excludePaths = append(excludePaths, "ignoredFile")

	os.Mkdir(path.Join(local, "testFolder"), 0755)
	os.Mkdir(path.Join(local, "testFolder2"), 0755)
	os.Mkdir(path.Join(local, "ignoredFolder"), 0755)
	excludePaths = append(excludePaths, "ignoredFolder")

	ioutil.WriteFile(path.Join(local, "testFolder", "testFile1"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFolder", "testFile2"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFolder", "ignoredFile"), []byte(fileContents), 0666)
	excludePaths = append(excludePaths, "testFolder/ignoredFile")

	ioutil.WriteFile(path.Join(local, "ignoredFolder", "testFile1"), []byte(fileContents), 0666)

	err := copyToContainerTestable(syncClient.Kubectl, syncClient.Pod, syncClient.Container, local, containerPath, excludePaths, true)
	if err != nil {
		t.Error(err)
		return
	}

	filesToCheck := []checkedFileOrFolder{
		checkedFileOrFolder{
			path:                "testFile1",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "testFile2",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "ignoredFile",
			shouldExistInLocal:  true,
			shouldExistInRemote: false,
		},
		checkedFileOrFolder{
			path:                "testFolder/testFile1",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "testFolder/testFile2",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "testFolder/ignoredFile",
			shouldExistInLocal:  true,
			shouldExistInRemote: false,
		},
		checkedFileOrFolder{
			path:                "ignoredFolder/testFile1",
			shouldExistInLocal:  true,
			shouldExistInRemote: false,
		},
	}
	foldersToCheck := []checkedFileOrFolder{
		checkedFileOrFolder{
			path:                "testFolder",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "testFolder2",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "ignoredFolder",
			shouldExistInLocal:  true,
			shouldExistInRemote: false,
		},
	}

	checkFilesAndFolders(t, filesToCheck, foldersToCheck, local, remote)

	// Check if there is an error in the error channel
	select {
	case err = <-syncClient.errorChan:
		t.Error(err)
		return
	default:
	}

}

type checkedFileOrFolder struct {
	path                string
	shouldExistInRemote bool
	shouldExistInLocal  bool
}

const fileContents = "TestContents"

func checkFilesAndFolders(t *testing.T, files []checkedFileOrFolder, folders []checkedFileOrFolder, local string, remote string) {

	timeout := 10 * time.Second
	beginTimeStamp := time.Now()

	var missingFileOrFolder checkedFileOrFolder

Outer:
	for time.Since(beginTimeStamp) < timeout {

		// Check files
		for _, v := range files {
			localFile := path.Join(local, v.path)
			remoteFile := path.Join(remote, v.path)

			_, err := os.Stat(localFile)
			if v.shouldExistInLocal && os.IsNotExist(err) {
				missingFileOrFolder = v
				continue Outer
			}
			if err != nil && !os.IsNotExist(err) {
				t.Error(err)
				return
			}
			if !v.shouldExistInLocal && !os.IsNotExist(err) {
				t.Error("Local File " + localFile + " shouldn't exist but it does")
				return
			}

			_, err = os.Stat(remoteFile)
			if v.shouldExistInRemote && os.IsNotExist(err) {
				missingFileOrFolder = v
				continue Outer
			}
			if err != nil && !os.IsNotExist(err) {
				t.Error(err)
				return
			}
			if !v.shouldExistInRemote && !os.IsNotExist(err) {
				t.Error("Remote File " + remoteFile + " shouldn't exist but it does")
				return
			}

			if v.shouldExistInLocal {
				data, err := ioutil.ReadFile(localFile)
				if err != nil {
					t.Error(err)
					return
				}
				if string(data) != fileContents {
					t.Errorf("Wrong file contents in file %s, got %s, expected %s", localFile, string(data), fileContents)
					return
				}
			}

			if v.shouldExistInRemote {
				data, err := ioutil.ReadFile(remoteFile)
				if err != nil {
					t.Error(err)
					return
				}
				if string(data) != fileContents {
					t.Errorf("Wrong file contentsin file %s, got %s, expected %s", remoteFile, string(data), fileContents)
					return
				}
			}
		}

		// Check folders
		for _, v := range folders {
			localFolder := path.Join(local, v.path)
			remoteFolder := path.Join(remote, v.path)

			stat, err := os.Stat(localFolder)
			if v.shouldExistInLocal && os.IsNotExist(err) {
				missingFileOrFolder = v
				continue Outer
			}
			if err != nil && !os.IsNotExist(err) {
				t.Error(err)
				return
			}
			if !v.shouldExistInLocal && !os.IsNotExist(err) {
				t.Error("Local Directory " + localFolder + " shouldn't exist but it does")
				return
			}
			if err == nil && stat.IsDir() == false {
				t.Errorf("Expected %s to be a dir", localFolder)
				return
			}

			stat, err = os.Stat(remoteFolder)
			if v.shouldExistInRemote && os.IsNotExist(err) {
				missingFileOrFolder = v
				continue Outer
			}
			if err != nil && !os.IsNotExist(err) {
				t.Error(err)
				return
			}
			if !v.shouldExistInRemote && !os.IsNotExist(err) {
				t.Error("Remote Directory " + remoteFolder + " shouldn't exist but it does")
				return
			}
			if err == nil && stat.IsDir() == false {
				t.Errorf("Expected %s to be a dir", remoteFolder)
				return
			}
		}

		//If this code is reached, everything is fine
		return

	}

	//If this code is reached, every time the results of the checks showed an unfinished sync. Timeout is reached

	printPathAndReturnNil := func(path string, f os.FileInfo, err error) error {
		t.Log(path)
		return nil
	}

	t.Log("Remote Path Content:")
	err := filepath.Walk(remote, printPathAndReturnNil)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("Local Path Content:")
	err = filepath.Walk(local, printPathAndReturnNil)
	if err != nil {
		t.Error(err)
		return
	}

	t.Error("Sync Failed. " +
		"Missing: " + path.Join(remote, missingFileOrFolder.path))
}
