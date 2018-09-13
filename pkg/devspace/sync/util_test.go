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

	remote, local, _ := initTestDirs(t)

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

	err := copyToContainerTestable(nil, nil, nil, local, remote, excludePaths, true)
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

	checkFilesAndFolders(t, filesToCheck, foldersToCheck, local, remote, 10*time.Second)

}

const (
	editInRemote = 0
	editInLocal  = 1
	editOutside  = 2
)

type checkedFileOrFolder struct {
	path                string
	shouldExistInRemote bool
	shouldExistInLocal  bool
	editLocation        int
}

type testCaseList []checkedFileOrFolder

func (arr testCaseList) Len() int {
	return len(arr)
}

func (arr testCaseList) Less(i, j int) bool {
	return len(arr[i].path) < len(arr[j].path)
}

func (arr testCaseList) Swap(i, j int) {
	x := arr[i]
	arr[i] = arr[j]
	arr[j] = x
}

const fileContents = "TestContents"

func checkFilesAndFolders(t *testing.T, files []checkedFileOrFolder, folders []checkedFileOrFolder, local string, remote string, timeout time.Duration) {

	beginTimeStamp := time.Now()

	var missingFileOrFolder string
	var unexpectedFileOrFolder string

Outer:
	for time.Since(beginTimeStamp) < timeout {

		missingFileOrFolder = ""
		unexpectedFileOrFolder = ""

		/*
			If something is expected to be there but it isn't, we expect that the sync-job isn't finished yet.
			The same applies if a file has missing content. Also if a file is there when it shouldn't be.
			In these cases we continue the outer Loop until everything is there or the time runs up.

			If something unexpected happens like an unxpected error or a wrong file type, we let the test fail and return
		*/
		// Check files
	FileCheck:
		for _, v := range files {
			localFile := path.Join(local, v.path)
			remoteFile := path.Join(remote, v.path)

			localData, err := ioutil.ReadFile(localFile)
			if v.shouldExistInLocal && os.IsNotExist(err) {
				missingFileOrFolder = localFile
				continue Outer
			}
			if !v.shouldExistInLocal && !os.IsNotExist(err) {
				unexpectedFileOrFolder = localFile
				continue Outer
			}
			if err != nil && !os.IsNotExist(err) {
				t.Error(err)
				return
			}

			remoteData, err := ioutil.ReadFile(remoteFile)
			if v.shouldExistInRemote && os.IsNotExist(err) {
				missingFileOrFolder = remoteFile
				continue Outer
			}
			if !v.shouldExistInRemote && !os.IsNotExist(err) {
				unexpectedFileOrFolder = remoteFile
				continue Outer
			}
			if !v.shouldExistInRemote && os.IsNotExist(err) {
				continue FileCheck
			}
			if err != nil {
				t.Error(err)
				return
			}

			if v.shouldExistInLocal {
				if string(localData) != fileContents {
					missingFileOrFolder = localFile
					continue Outer
				}
			}

			if v.shouldExistInRemote {
				if string(remoteData) != fileContents {
					missingFileOrFolder = remoteFile
					continue Outer
				}
			}
		}

		// Check folders
		for _, v := range folders {
			localFolder := path.Join(local, v.path)
			remoteFolder := path.Join(remote, v.path)

			stat, err := os.Stat(localFolder)
			if v.shouldExistInLocal && os.IsNotExist(err) {
				missingFileOrFolder = localFolder
				continue Outer
			}
			if !v.shouldExistInLocal && !os.IsNotExist(err) {
				unexpectedFileOrFolder = localFolder
				continue Outer
			}
			if err != nil && !os.IsNotExist(err) {
				t.Error(err)
				return
			}
			if err == nil && stat.IsDir() == false {
				t.Errorf("Expected %s to be a dir", localFolder)
				return
			}

			stat, err = os.Stat(remoteFolder)
			if v.shouldExistInRemote && os.IsNotExist(err) {
				missingFileOrFolder = remoteFolder
				continue Outer
			}
			if !v.shouldExistInRemote && !os.IsNotExist(err) {
				unexpectedFileOrFolder = remoteFolder
				continue Outer
			}
			if err != nil && !os.IsNotExist(err) {
				t.Error(err)
				return
			}
			if err == nil && stat.IsDir() == false {
				t.Errorf("Expected %s to be a dir", remoteFolder)
				return
			}
		}

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

	if missingFileOrFolder != "" {
		t.Error("Sync Failed. Missing: " + missingFileOrFolder)
	} else if unexpectedFileOrFolder != "" {
		t.Error("Sync Failed. Shouldn't be there: " + unexpectedFileOrFolder)
	} else {
		t.Error("unexpected")
	}
}
