package sync

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/juju/errors"
)

func initTestDirs(t *testing.T) (string, string, string) {
	testRemotePath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Couldn't create test dir: %v", err)
	}

	testLocalPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Couldn't create test dir: %v", err)
	}

	outside, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Couldn't create test dir: %v", err)
	}

	return testRemotePath, testLocalPath, outside
}

func createTestSyncClient(testLocalPath, testRemotePath string) *SyncConfig {
	syncLog = log.GetInstance()

	return &SyncConfig{
		WatchPath: testLocalPath,
		DestPath:  testRemotePath,

		testing: true,
		verbose: true,
	}
}

func TestInitialSync(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on windows")
	}

	remote, local, outside := initTestDirs(t)
	defer os.RemoveAll(remote)
	defer os.RemoveAll(local)

	syncClient := createTestSyncClient(local, remote)
	defer syncClient.Stop()

	syncClient.errorChan = make(chan error)
	setExcludePaths(syncClient)

	// Start client
	err := syncClient.setup()
	if err != nil {
		t.Errorf("Couldn't init test sync client: %v", err)
		return
	}

	// Start upstream
	err = syncClient.upstream.start()
	if err != nil {
		t.Error(err)
		return
	}

	// Start downstream
	err = syncClient.downstream.start()
	if err != nil {
		t.Error(err)
		return
	}

	filesToCheck, foldersToCheck := makeBasicTestCases()

	err = createTestFilesAndFolders(local, remote, outside, filesToCheck, foldersToCheck)
	if err != nil {
		t.Error(err)
		return
	}

	go syncClient.startUpstream()

	// Do initial sync
	err = syncClient.initialSync()
	if err != nil {
		t.Error(err)
		return
	}

	checkFilesAndFolders(t, filesToCheck, foldersToCheck, local, remote, 15*time.Second)
}
func TestNormalSync(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on windows")
	}

	remote, local, outside := initTestDirs(t)
	defer os.RemoveAll(remote)
	defer os.RemoveAll(local)
	defer os.RemoveAll(outside)

	syncClient := createTestSyncClient(local, remote)
	defer syncClient.Stop()

	syncClient.errorChan = make(chan error)
	setExcludePaths(syncClient)

	// Start client
	err := syncClient.setup()
	if err != nil {
		t.Errorf("Couldn't init test sync client: %v", err)
		return
	}

	// Start upstream
	err = syncClient.upstream.start()
	if err != nil {
		t.Error(err)
		return
	}

	// Start downstream
	err = syncClient.downstream.start()
	if err != nil {
		t.Error(err)
		return
	}

	syncClient.readyChan = make(chan bool)

	go syncClient.startUpstream()
	go syncClient.startDownstream()

	<-syncClient.readyChan

	filesToCheck, foldersToCheck := makeBasicTestCases()

	err = createTestFilesAndFolders(local, remote, outside, filesToCheck, foldersToCheck)
	if err != nil {
		t.Error(err)
		return
	}
	checkFilesAndFolders(t, filesToCheck, foldersToCheck, local, remote, 10*time.Second)

	filesToCheck, foldersToCheck, err = removeSomeTestFilesAndFolders(local, remote, filesToCheck, foldersToCheck, "_Remove")
	if err != nil {
		t.Error(err)
	}
	checkFilesAndFolders(t, filesToCheck, foldersToCheck, local, remote, 10*time.Second)

}

func setExcludePaths(syncClient *SyncConfig) {

	syncClient.ExcludePaths = []string{
		"ignoreFileLocal",
		"ignoreFolderLocal",
		"testFolder/ignoreFileLocal",
		"ignoreFileRemote",
		"ignoreFolderRemote",
		"testFolder/ignoreFileRemote",
	}

	syncClient.DownloadExcludePaths = []string{
		"noDownloadFileLocal",
		"noDownloadFolderLocal",
		"testFolder/noDownloadFileLocal",
		"noDownloadFileRemote",
		"noDownloadFolderRemote",
		"testFolder/noDownloadFileRemote",
	}

	syncClient.UploadExcludePaths = []string{
		"noUploadFileLocal",
		"noUploadFolderLocal",
		"testFolder/noUploadFileLocal",
		"noUploadFileRemote",
		"noUploadFolderRemote",
		"testFolder/noUploadFileRemote",
	}

	for _, excludeArray := range [3][]string{syncClient.ExcludePaths, syncClient.DownloadExcludePaths, syncClient.UploadExcludePaths} {
		for _, ignoreString := range excludeArray {
			excludeArray = append(excludeArray, ignoreString+"_Remove")
			excludeArray = append(excludeArray, ignoreString+"_RenameToFullContext")
		}
	}

	syncClient.initIgnoreParsers()

}

func makeBasicTestCases() ([]checkedFileOrFolder, []checkedFileOrFolder) {
	filesToCheck := []checkedFileOrFolder{
		checkedFileOrFolder{
			path:                "testFileLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
			editLocation:        editInLocal,
		},
		checkedFileOrFolder{
			path:                "ignoreFileLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: false,
			editLocation:        editInLocal,
		},
		checkedFileOrFolder{
			path:                "noDownloadFileLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
			editLocation:        editInLocal,
		},
		checkedFileOrFolder{
			path:                "noUploadFileLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: false,
			editLocation:        editInLocal,
		},
	}

	foldersToCheck := []checkedFileOrFolder{
		checkedFileOrFolder{
			path:                "testFolder",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
			editLocation:        editInLocal,
		},
		checkedFileOrFolder{
			path:                "testFolderLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
			editLocation:        editInLocal,
		},
		checkedFileOrFolder{
			path:                "ignoreFolderLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: false,
			editLocation:        editInLocal,
		},
		checkedFileOrFolder{
			path:                "noDownloadFolderLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
			editLocation:        editInLocal,
		},
		checkedFileOrFolder{
			path:                "noUploadFolderLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: false,
			editLocation:        editInLocal,
		},
	}

	//Add Files and Folders that are editd in Remote
	for n, array := range [2][]checkedFileOrFolder{filesToCheck, foldersToCheck} {
		for _, f := range array {

			if strings.Contains(f.path, "Upload") {
				f.path = strings.Replace(f.path, "Upload", "Download", -1)
			} else if strings.Contains(f.path, "Download") {
				f.path = strings.Replace(f.path, "Download", "Upload", -1)
			}

			remoteEquivalent := checkedFileOrFolder{
				path:                strings.Replace(f.path, "Local", "Remote", -1),
				shouldExistInLocal:  f.shouldExistInRemote,
				shouldExistInRemote: f.shouldExistInLocal,
				editLocation:        editInRemote,
			}
			array = append(array, remoteEquivalent)
		}
		if n == 0 {
			filesToCheck = array
		} else {
			foldersToCheck = array
		}
	}

	//Add Files and Folders that are inside a shared testFolder
	for n, array := range [2][]checkedFileOrFolder{filesToCheck, foldersToCheck} {

		for _, f := range array {
			deepEquivalent := checkedFileOrFolder{
				path:                path.Join("testFolder/", f.path),
				shouldExistInLocal:  f.shouldExistInLocal,
				shouldExistInRemote: f.shouldExistInRemote,
				editLocation:        f.editLocation,
			}
			array = append(array, deepEquivalent)
		}

		if n == 0 {
			filesToCheck = array
		} else {
			foldersToCheck = array
		}
	}

	return filesToCheck, foldersToCheck
}

func createTestFilesAndFolders(local string, remote string, outside string, filesToCheck []checkedFileOrFolder, foldersToCheck []checkedFileOrFolder) error {

	for _, f := range foldersToCheck {
		var parentDir string
		if f.editLocation == editInLocal {
			parentDir = local
		} else if f.editLocation == editInRemote {
			parentDir = remote
		} else if f.editLocation == editOutside {
			parentDir = outside
		} else {
			return errors.New("CreateLocation " + string(f.editLocation) + " unknown")
		}
		os.Mkdir(path.Join(parentDir, f.path), 0755)
	}

	for _, f := range filesToCheck {
		var parentDir string
		if f.editLocation == editInLocal {
			parentDir = local
		} else if f.editLocation == editInRemote {
			parentDir = remote
		} else if f.editLocation == editOutside {
			parentDir = outside
		} else {
			return errors.New("CreateLocation " + string(f.editLocation) + " unknown")
		}
		ioutil.WriteFile(path.Join(parentDir, f.path), []byte(fileContents), 0666)

	}

	/*
		//Add Remove-Stuff
		for _, f := range filesToCheck {
			removeEquivalent := checkedFileOrFolder{
				path:                f.path + "_Remove",
				shouldExistInLocal:  f.shouldExistInLocal,
				shouldExistInRemote: f.shouldExistInRemote,
			}
			filesToCheck = append(filesToCheck, removeEquivalent)
		}
		for _, f := range foldersToCheck {
			if f.path == "testFolder" {
				continue
			}
			removeEquivalent := checkedFileOrFolder{
				path:                f.path + "_Remove",
				shouldExistInLocal:  f.shouldExistInLocal,
				shouldExistInRemote: f.shouldExistInRemote,
			}
			foldersToCheck = append(foldersToCheck, removeEquivalent)
		}

		//Add Rename-Stuff
		for _, f := range filesToCheck {
			if strings.HasSuffix(f.path, "_Remove") {
				continue
			}
			renameEquivalent := checkedFileOrFolder{
				path:                f.path + "_RenameToFullContext",
				shouldExistInLocal:  f.shouldExistInLocal,
				shouldExistInRemote: f.shouldExistInRemote,
			}
			filesToCheck = append(filesToCheck, renameEquivalent)

			if strings.Contains(f.path, "testFile") {
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToOutside",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
				}
				filesToCheck = append(filesToCheck, renameEquivalent)
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToIgnore",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
				}
				filesToCheck = append(filesToCheck, renameEquivalent)
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToNoDownload",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
				}
				filesToCheck = append(filesToCheck, renameEquivalent)
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToNoUpload",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
				}
				filesToCheck = append(filesToCheck, renameEquivalent)
			}
		}

		for _, f := range foldersToCheck {
			if strings.HasSuffix(f.path, "_Remove") {
				continue
			}
			if f.path == "testFolder" {
				continue
			}
			renameEquivalent := checkedFileOrFolder{
				path:                f.path + "_RenameToFullContext",
				shouldExistInLocal:  f.shouldExistInLocal,
				shouldExistInRemote: f.shouldExistInRemote,
			}
			foldersToCheck = append(foldersToCheck, renameEquivalent)

			if strings.Contains(f.path, "testFile") {
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToOutside",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
				}
				foldersToCheck = append(foldersToCheck, renameEquivalent)
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToIgnore",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
				}
				foldersToCheck = append(foldersToCheck, renameEquivalent)
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToNoDownload",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
				}
				foldersToCheck = append(foldersToCheck, renameEquivalent)
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToNoUpload",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
				}
				foldersToCheck = append(foldersToCheck, renameEquivalent)
			}
		}
	*/

	return nil
}

func removeSomeTestFilesAndFolders(local string, remote string, filesToCheck []checkedFileOrFolder, foldersToCheck []checkedFileOrFolder, removeSuffix string) ([]checkedFileOrFolder, []checkedFileOrFolder, error) {

	var completeSuffix string
	removeIfSuffixMatch := func(path string, f os.FileInfo, err error) error {
		if strings.HasSuffix(path, completeSuffix) {
			return os.RemoveAll(path)
		}
		return nil
	}

	completeSuffix = "Remote" + removeSuffix
	err := filepath.Walk(remote, removeIfSuffixMatch)
	if err != nil {
		return nil, nil, err
	}
	completeSuffix = "Local" + removeSuffix
	err = filepath.Walk(local, removeIfSuffixMatch)
	if err != nil {
		return nil, nil, err
	}

	for n, f := range filesToCheck {
		if strings.HasSuffix(f.path, removeSuffix) {
			filesToCheck[n].shouldExistInLocal = false
			filesToCheck[n].shouldExistInRemote = false
		}
	}

	for n, f := range foldersToCheck {
		if strings.HasSuffix(f.path, removeSuffix) {
			foldersToCheck[n].shouldExistInLocal = false
			foldersToCheck[n].shouldExistInRemote = false
		}
	}

	return filesToCheck, foldersToCheck, nil
}

func TestCreateDirInFileMap(t *testing.T) {
	sync := SyncConfig{
		fileIndex: newFileIndex(),
	}

	sync.fileIndex.CreateDirInFileMap("/TestDir1/TestDir2/TestDir3/TestDir4")

	if len(sync.fileIndex.fileMap) != 4 {
		t.Error("Create dir in file map failed!")
		t.Fail()
	}
}

func TestRemoveDirInFileMap(t *testing.T) {
	sync := SyncConfig{
		fileIndex: newFileIndex(),
	}

	sync.fileIndex.fileMap = map[string]*fileInformation{
		"/TestDir": {
			Name:        "/TestDir",
			IsDirectory: true,
		},
		"/TestDir/File1": {
			Name:        "/TestDir/File1",
			Size:        1234,
			Mtime:       1234,
			IsDirectory: false,
		},
		"/TestDir2": {
			Name:        "/TestDir2",
			IsDirectory: true,
		},
	}

	sync.fileIndex.RemoveDirInFileMap("/TestDir")

	if len(sync.fileIndex.fileMap) != 1 {
		t.Error("Remove dir in file map failed!")
		t.Fail()
	}
}

func TestCeilMtime(t *testing.T) {
	ceiledNumberNano := time.Unix(1533647574, 0)
	ceiledNumberSeconds := time.Unix(1533647574, 0)

	unceiledNumberNano := time.Unix(1533647574, 1)
	unceiledNumberSeconds := time.Unix(1533647575, 0)

	if ceilMtime(ceiledNumberNano) != ceiledNumberSeconds.Unix() {
		t.Error("ceilMtime failed ceiledNumberNano != ceiledNumberSeconds")
		t.Fail()
	}

	if ceilMtime(unceiledNumberNano) != unceiledNumberSeconds.Unix() {
		t.Error("ceilMtime failed unceiledNumberNano != unceiledNumberSeconds")
		t.Fail()
	}
}
