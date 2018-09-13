package sync

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
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

	filesToCheck, foldersToCheck := makeBasicTestCases()

	syncClient := createTestSyncClient(local, remote)
	defer syncClient.Stop()

	syncClient.errorChan = make(chan error)
	setExcludePaths(syncClient, append(filesToCheck, foldersToCheck...))

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

/*
func TestNormalSync(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on windows")
	}

	remote, local, outside := initTestDirs(t)
	defer os.RemoveAll(remote)
	defer os.RemoveAll(local)
	defer os.RemoveAll(outside)

	filesToCheck, foldersToCheck := makeBasicTestCases()
	filesToCheck, foldersToCheck = makeRemoveAndRenameTestCases(filesToCheck, foldersToCheck)
	sort.Stable(foldersToCheck)

	syncClient := createTestSyncClient(local, remote)
	defer syncClient.Stop()

	syncClient.errorChan = make(chan error)
	setExcludePaths(syncClient, append(filesToCheck, foldersToCheck...))

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

	err = createTestFilesAndFolders(local, remote, outside, filesToCheck, foldersToCheck)
	if err != nil {
		t.Error(err)
		return
	}
	checkFilesAndFolders(t, filesToCheck, foldersToCheck, local, remote, 10*time.Second)

	filesToCheck, foldersToCheck, err = removeSomeTestFilesAndFolders(local, remote, filesToCheck, foldersToCheck, "_Remove")
	if err != nil {
		t.Error(err)
		return
	}
	checkFilesAndFolders(t, filesToCheck, foldersToCheck, local, remote, 10*time.Second)

	filesToCheck, foldersToCheck, err = renameSomeTestFilesAndFolders(local, remote, outside, filesToCheck, foldersToCheck)
	if err != nil {
		t.Error(err)
		return
	}
	checkFilesAndFolders(t, filesToCheck, foldersToCheck, local, remote, 10*time.Second)

}
*/

func setExcludePaths(syncClient *SyncConfig, testCases testCaseList) {

	syncClient.ExcludePaths = []string{}

	syncClient.DownloadExcludePaths = []string{}

	syncClient.UploadExcludePaths = []string{}

	for _, testCase := range testCases {
		/*
			All paths that should be in these ExcludePaths are marked like this with these strings.
			for Example: ignoreFileLocal
			The RenameTo... parts of some files contain those, too, but those use Big Letters so they are not excluded.
			For example: testFileLocal_RenameToIgnore
		*/
		if strings.Contains(testCase.path, "ignore") {
			syncClient.ExcludePaths = append(syncClient.ExcludePaths, testCase.path)
		} else if strings.Contains(testCase.path, "noDownload") {
			syncClient.DownloadExcludePaths = append(syncClient.DownloadExcludePaths, testCase.path)
		} else if strings.Contains(testCase.path, "noUpload") {
			syncClient.UploadExcludePaths = append(syncClient.UploadExcludePaths, testCase.path)
		} else if strings.HasSuffix(testCase.path, "_RenameToIgnore") {
			syncClient.ExcludePaths = append(syncClient.ExcludePaths, testCase.path+"After")
		} else if strings.HasSuffix(testCase.path, "_RenameToNoDownload") {
			syncClient.DownloadExcludePaths = append(syncClient.DownloadExcludePaths, testCase.path+"After")
		} else if strings.HasSuffix(testCase.path, "_RenameToNoUpload") {
			syncClient.UploadExcludePaths = append(syncClient.UploadExcludePaths, testCase.path+"After")
		}
	}

	syncClient.initIgnoreParsers()

}

func makeBasicTestCases() (testCaseList, testCaseList) {
	filesToCheck := testCaseList{
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

	foldersToCheck := testCaseList{
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

	//Add Files and Folders that are edited in Remote
	filesToCheck = makeRemoteTestCases(filesToCheck)
	foldersToCheck = makeRemoteTestCases(foldersToCheck)

	//Add Files and Folders that are inside a shared testFolder
	filesToCheck = makeDeepTestCases(filesToCheck)
	foldersToCheck = makeDeepTestCases(foldersToCheck)

	return filesToCheck, foldersToCheck
}

func makeRemoveAndRenameTestCases(filesToCheck testCaseList, foldersToCheck testCaseList) (testCaseList, testCaseList) {

	for n, array := range [2]testCaseList{filesToCheck, foldersToCheck} {
		for _, f := range array {

			if f.path == "testFolder" {
				continue
			}

			removeEquivalent := checkedFileOrFolder{
				path:                f.path + "_Remove",
				shouldExistInLocal:  f.shouldExistInLocal,
				shouldExistInRemote: f.shouldExistInRemote,
				editLocation:        f.editLocation,
			}
			array = append(array, removeEquivalent)

			renameEquivalent := checkedFileOrFolder{
				path:                f.path + "_RenameToFullContext",
				shouldExistInLocal:  f.shouldExistInLocal,
				shouldExistInRemote: f.shouldExistInRemote,
				editLocation:        f.editLocation,
			}
			array = append(array, renameEquivalent)

			isFullyIncluded, _ := regexp.Compile("(testFolder\\/)?(testFile|testFolder)(Local|Remote)$")

			if isFullyIncluded.MatchString(f.path) {
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToOutside",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
					editLocation:        f.editLocation,
				}
				array = append(array, renameEquivalent)
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToIgnore",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
					editLocation:        f.editLocation,
				}
				array = append(array, renameEquivalent)
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToNoDownload",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
					editLocation:        f.editLocation,
				}
				array = append(array, renameEquivalent)
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToNoUpload",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
					editLocation:        f.editLocation,
				}
				array = append(array, renameEquivalent)
			}
		}

		if n == 0 {
			filesToCheck = array
		} else {
			foldersToCheck = array
		}
	}

	renameFilesFromOutside := testCaseList{
		checkedFileOrFolder{
			path:                "testFileOutsideToLocal_RenameToFullContext",
			shouldExistInLocal:  false,
			shouldExistInRemote: false,
			editLocation:        editOutside,
		},
		checkedFileOrFolder{
			path:                "testFileOutsideToRemote_RenameToFullContext",
			shouldExistInLocal:  false,
			shouldExistInRemote: false,
			editLocation:        editOutside,
		},
	}

	renameFolderFromOutside := testCaseList{
		checkedFileOrFolder{
			path:                "testFolderOutsideToLocal_RenameToFullContext",
			shouldExistInLocal:  false,
			shouldExistInRemote: false,
			editLocation:        editOutside,
		},
		checkedFileOrFolder{
			path:                "testFolderOutsideToRemote_RenameToFullContext",
			shouldExistInLocal:  false,
			shouldExistInRemote: false,
			editLocation:        editOutside,
		},
		checkedFileOrFolder{
			path:                "testFolder",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
			editLocation:        editOutside,
		},
	}

	renameFilesFromOutside = makeDeepTestCases(renameFilesFromOutside)
	renameFolderFromOutside = makeDeepTestCases(renameFolderFromOutside)

	filesToCheck = append(filesToCheck, renameFilesFromOutside...)
	foldersToCheck = append(foldersToCheck, renameFolderFromOutside...)

	return filesToCheck, foldersToCheck

}

func makeRemoteTestCases(testCases testCaseList) testCaseList {

	for _, f := range testCases {

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
		testCases = append(testCases, remoteEquivalent)
	}

	return testCases
}

func makeDeepTestCases(testCases testCaseList) testCaseList {

	for _, f := range testCases {

		if f.path == "testFolder" {
			continue
		}

		deepEquivalent := checkedFileOrFolder{
			path:                path.Join("testFolder/", f.path),
			shouldExistInLocal:  f.shouldExistInLocal,
			shouldExistInRemote: f.shouldExistInRemote,
			editLocation:        f.editLocation,
		}
		testCases = append(testCases, deepEquivalent)
	}

	return testCases
}

func createTestFilesAndFolders(local string, remote string, outside string, filesToCheck testCaseList, foldersToCheck testCaseList) error {

	for _, f := range foldersToCheck {
		parentDir, err := getParentDir(local, remote, outside, f.editLocation)
		if err != nil {
			return errors.Trace(err)
		}

		err = os.Mkdir(path.Join(parentDir, f.path), 0755)
		if err != nil {
			return errors.Trace(err)
		}
	}

	for _, f := range filesToCheck {
		parentDir, err := getParentDir(local, remote, outside, f.editLocation)
		if err != nil {
			return errors.Trace(err)
		}

		err = ioutil.WriteFile(path.Join(parentDir, f.path), []byte(fileContents), 0666)
		if err != nil {
			return errors.Trace(err)
		}

	}

	return nil
}

func removeSomeTestFilesAndFolders(local string, remote string, filesToCheck testCaseList, foldersToCheck testCaseList, removeSuffix string) (testCaseList, testCaseList, error) {

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

func renameSomeTestFilesAndFolders(local string, remote string, outside string, filesToCheck testCaseList, foldersToCheck testCaseList) (testCaseList, testCaseList, error) {

	for n, array := range [2]testCaseList{filesToCheck, foldersToCheck} {

		for n, f := range array {

			if !strings.Contains(f.path, "_Rename") {
				continue
			}

			fromParentDir, err := getParentDir(local, remote, outside, f.editLocation)
			if err != nil {
				return nil, nil, errors.Trace(err)
			}
			fromPath := path.Join(fromParentDir, f.path)

			var toParentDir string
			if strings.HasSuffix(f.path, "_RenameToOutside") {
				toParentDir = outside
			} else if strings.Contains(f.path, "Local_Rename") {
				toParentDir = local
			} else if strings.Contains(f.path, "Remote_Rename") {
				toParentDir = remote
			}

			f.path = f.path + "After"
			toPath := path.Join(toParentDir, f.path)

			err = os.Rename(fromPath, toPath)
			if err != nil {
				return nil, nil, errors.Trace(err)
			}

			if strings.HasSuffix(f.path, "_RenameToFullContextAfter") {
				f.shouldExistInLocal = true
				f.shouldExistInRemote = true
			} else if strings.HasSuffix(f.path, "_RenameToNoDownloadAfter") {
				f.shouldExistInRemote = true
				f.shouldExistInLocal = f.editLocation == editInLocal
			} else if strings.HasSuffix(f.path, "_RenameToNoUploadAfter") {
				f.shouldExistInLocal = true
				f.shouldExistInRemote = f.editLocation == editInRemote
			} else if strings.HasSuffix(f.path, "_RenameToIgnoreAfter") {
				f.shouldExistInLocal = f.editLocation == editInLocal
				f.shouldExistInRemote = f.editLocation == editInRemote
			} else if strings.HasSuffix(f.path, "_RenameToOutsideAfter") {
				f.shouldExistInLocal = false
				f.shouldExistInRemote = false
			} else {
				return nil, nil, errors.New("Bad rename suffix of " + f.path)
			}

			array[n] = f

		}

		if n == 0 {
			filesToCheck = array
		} else {
			foldersToCheck = array
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
