//go:build !windows
// +build !windows

package sync

import (
	"os"
	"path"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/pkg/errors"
)

const (
	editInRemote   = 0
	editInLocal    = 1
	editOutside    = 2
	editSymLinkDir = 3
)

func getParentDir(localDir string, remoteDir string, outsideDir string, editLocation int) (string, error) {
	if editLocation == editInLocal {
		return localDir, nil
	} else if editLocation == editInRemote {
		return remoteDir, nil
	} else if editLocation == editOutside {
		return outsideDir, nil
	} else if editLocation == editSymLinkDir {
		return filepath.Join(outsideDir, "symlinkTargets"), nil
	}

	return "", errors.New("CreateLocation " + strconv.Itoa(editLocation) + " unknown")
}

type checkedFileOrFolder struct {
	path                string
	shouldExistInRemote bool
	shouldExistInLocal  bool
	editLocation        int
	isSymLink           bool
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
		time.Sleep(time.Millisecond * 100)

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

			localData, err := os.ReadFile(localFile)
			if v.shouldExistInLocal && os.IsNotExist(err) {
				missingFileOrFolder = localFile
				continue Outer
			}
			if !v.shouldExistInLocal && !os.IsNotExist(err) {
				unexpectedFileOrFolder = localFile
				continue Outer
			}
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}

			remoteData, err := os.ReadFile(remoteFile)
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
				t.Fatal(err)
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
				t.Fatal(err)
			}
			if err == nil && stat.IsDir() == false {
				t.Fatalf("Expected %s to be a dir", localFolder)
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
				t.Fatalf("Expected %s to be a dir", remoteFolder)
			}
		}

		//If this code is reached, everything is fine
		return
	}

	// Print time since check start
	t.Logf("Time since check start: %s", time.Since(beginTimeStamp).String())

	//If this code is reached, every time the results of the checks showed an unfinished sync. Timeout is reached
	printPathAndReturnNil := func(path string, f os.FileInfo, err error) error {
		t.Log(path)
		return nil
	}

	t.Log("Remote Path Content:")
	err := filepath.Walk(remote, printPathAndReturnNil)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Local Path Content:")
	err = filepath.Walk(local, printPathAndReturnNil)
	if err != nil {
		t.Fatal(err)
	}

	if missingFileOrFolder != "" {
		t.Fatal("Sync Failed. Missing: " + missingFileOrFolder)
	} else if unexpectedFileOrFolder != "" {
		t.Fatal("Sync Failed. Shouldn't be there: " + unexpectedFileOrFolder)
	} else {
		t.Fatal("unexpected")
	}
}
