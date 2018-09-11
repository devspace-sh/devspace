package sync

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/covexo/devspace/pkg/util/log"
)

func initTestDirs(t *testing.T) (string, string) {
	testRemotePath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Couldn't create test dir: %v", err)
	}

	testLocalPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Couldn't create test dir: %v", err)
	}

	return testRemotePath, testLocalPath
}

func createTestSyncClient(testLocalPath, testRemotePath string) *SyncConfig {
	syncLog = log.GetInstance()

	return &SyncConfig{
		WatchPath: testLocalPath,
		DestPath:  testRemotePath,

		testing: true,
	}
}

func removeFolderAndWait(from, to, postfix string) error {
	foldernameFrom := path.Join(from, "testFolder"+postfix)
	foldernameTo := path.Join(to, "testFolder"+postfix)

	os.RemoveAll(foldernameFrom)

	for i := 0; i < 50; i++ {
		if _, err := os.Stat(foldernameTo); err != nil {
			return nil
		}

		time.Sleep(time.Millisecond * 100)
	}

	return fmt.Errorf("Removing folder %s wasn't correctly synced to %s", foldernameFrom, foldernameTo)
}

func removeFileAndWait(from, to, postfix string) error {
	filenameFrom := path.Join(from, "testFile"+postfix)
	filenameTo := path.Join(to, "testFile"+postfix)

	os.Remove(filenameFrom)

	for i := 0; i < 50; i++ {
		if _, err := os.Stat(filenameTo); err != nil {
			return nil
		}

		time.Sleep(time.Millisecond * 100)
	}

	return fmt.Errorf("Removing file %s wasn't correctly synced to %s", filenameFrom, filenameTo)
}

func createFolderAndWait(from, to, postfix string) error {
	foldernameFrom := path.Join(from, "testFolder"+postfix)
	foldernameTo := path.Join(to, "testFolder"+postfix)

	os.Mkdir(foldernameFrom, 0755)

	for i := 0; i < 50; i++ {
		if stat, err := os.Stat(foldernameTo); err == nil {
			if stat.IsDir() == false {
				return fmt.Errorf("Created folder %s from is a file in destination %s", foldernameFrom, foldernameTo)
			}

			return nil
		}

		time.Sleep(time.Millisecond * 100)
	}

	return fmt.Errorf("Created folder %s wasn't correctly synced to %s", foldernameFrom, foldernameTo)
}

func createFileAndWait(from, to, postfix string) error {
	filenameFrom := path.Join(from, "testFile"+postfix)
	filenameTo := path.Join(to, "testFile"+postfix)
	fileContents := "testFile" + postfix

	ioutil.WriteFile(filenameFrom, []byte(fileContents), 0666)

	for i := 0; i < 50; i++ {
		time.Sleep(time.Millisecond * 100)

		if _, err := os.Stat(filenameTo); err == nil {
			data, err := ioutil.ReadFile(filenameTo)
			if err != nil {
				continue
			}
			if string(data) != fileContents {
				continue
			}

			return nil
		}
	}

	return fmt.Errorf("Created file %s wasn't correctly synced to %s", filenameFrom, filenameTo)
}

func TestInitialSync(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on windows")
	}

	remote, local := initTestDirs(t)
	defer os.RemoveAll(remote)
	defer os.RemoveAll(local)

	syncClient := createTestSyncClient(local, remote)
	defer syncClient.Stop()

	syncClient.errorChan = make(chan error)

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

	// Create test files
	fileContents := "TestContents"

	// Write local files
	ioutil.WriteFile(path.Join(local, "testFile1"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFile2"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "ignoreFileLocal"), []byte(fileContents), 0666)
	syncClient.ExcludePaths = append(syncClient.ExcludePaths, "ignoreFileLocal")
	ioutil.WriteFile(path.Join(local, "noDownloadFileLocal"), []byte(fileContents), 0666)
	syncClient.DownloadExcludePaths = append(syncClient.DownloadExcludePaths, "noDownloadFileLocal")
	ioutil.WriteFile(path.Join(local, "noUploadFileLocal"), []byte(fileContents), 0666)
	syncClient.UploadExcludePaths = append(syncClient.UploadExcludePaths, "noUploadFileLocal")

	os.Mkdir(path.Join(local, "testFolder"), 0755)
	os.Mkdir(path.Join(local, "testFolder2"), 0755)
	os.Mkdir(path.Join(local, "ignoreFolderLocal"), 0755)
	syncClient.ExcludePaths = append(syncClient.ExcludePaths, "ignoreFolderLocal")
	os.Mkdir(path.Join(local, "noDownloadFolderLocal"), 0755)
	syncClient.DownloadExcludePaths = append(syncClient.DownloadExcludePaths, "noDownloadFolderLocal")
	os.Mkdir(path.Join(local, "noUploadFolderLocal"), 0755)
	syncClient.UploadExcludePaths = append(syncClient.UploadExcludePaths, "noUploadFolderLocal")

	ioutil.WriteFile(path.Join(local, "testFolder", "testFile1"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFolder", "testFile2"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFolder", "ignoreFileLocal"), []byte(fileContents), 0666)
	syncClient.ExcludePaths = append(syncClient.ExcludePaths, "testFolder/ignoreFileLocal")
	ioutil.WriteFile(path.Join(local, "testFolder", "noDownloadFileLocal"), []byte(fileContents), 0666)
	syncClient.DownloadExcludePaths = append(syncClient.DownloadExcludePaths, "testFolder/noDownloadFileLocal")
	ioutil.WriteFile(path.Join(local, "testFolder", "noUploadFileLocal"), []byte(fileContents), 0666)
	syncClient.UploadExcludePaths = append(syncClient.UploadExcludePaths, "testFolder/noUploadFileLocal")

	// Write remote files
	ioutil.WriteFile(path.Join(remote, "testFile3"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "testFile4"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "ignoreFileRemote"), []byte(fileContents), 0666)
	syncClient.ExcludePaths = append(syncClient.ExcludePaths, "ignoreFileRemote")
	ioutil.WriteFile(path.Join(remote, "noDownloadFileRemote"), []byte(fileContents), 0666)
	syncClient.DownloadExcludePaths = append(syncClient.DownloadExcludePaths, "noDownloadFileRemote")
	ioutil.WriteFile(path.Join(remote, "noUploadFileRemote"), []byte(fileContents), 0666)
	syncClient.UploadExcludePaths = append(syncClient.UploadExcludePaths, "noUploadFileRemote")

	os.Mkdir(path.Join(remote, "testFolder"), 0755)
	os.Mkdir(path.Join(remote, "testFolder3"), 0755)
	os.Mkdir(path.Join(remote, "ignoreFolderRemote"), 0755)
	syncClient.ExcludePaths = append(syncClient.ExcludePaths, "ignoreFolderRemote")
	os.Mkdir(path.Join(remote, "noDownloadFolderRemote"), 0755)
	syncClient.DownloadExcludePaths = append(syncClient.DownloadExcludePaths, "noDownloadFolderRemote")
	os.Mkdir(path.Join(remote, "noUploadFolderRemote"), 0755)
	syncClient.UploadExcludePaths = append(syncClient.UploadExcludePaths, "noUploadFolderRemote")

	ioutil.WriteFile(path.Join(remote, "testFolder", "testFile3"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "testFolder", "testFile4"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "ignoreFileRemote"), []byte(fileContents), 0666)
	syncClient.ExcludePaths = append(syncClient.ExcludePaths, "ignoreFileRemote")
	ioutil.WriteFile(path.Join(remote, "noDownloadFileRemote"), []byte(fileContents), 0666)
	syncClient.DownloadExcludePaths = append(syncClient.DownloadExcludePaths, "noDownloadFileRemote")
	ioutil.WriteFile(path.Join(remote, "noUploadFileRemote"), []byte(fileContents), 0666)
	syncClient.UploadExcludePaths = append(syncClient.UploadExcludePaths, "noUploadFileRemote")

	go syncClient.startUpstream()

	// Do initial sync
	err = syncClient.initialSync()
	if err != nil {
		t.Error(err)
		return
	}

	// Check outcome
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
			path:                "ignoreFileLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: false,
		},
		checkedFileOrFolder{
			path:                "noDownloadFileLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "noUploadFileLocal",
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
			path:                "testFolder/ignoreFileLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: false,
		},
		checkedFileOrFolder{
			path:                "testFolder/noDownloadFileLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "testFolder/noUploadFileLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: false,
		},

		checkedFileOrFolder{
			path:                "testFile3",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "testFile4",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "ignoreFileRemote",
			shouldExistInLocal:  false,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "noDownloadFileRemote",
			shouldExistInLocal:  false,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "noUploadFileRemote",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "testFolder/testFile3",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "testFolder/testFile4",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "testFolder/ignoreFileRemote",
			shouldExistInLocal:  false,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "testFolder/noDownloadFileRemote",
			shouldExistInLocal:  false,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "testFolder/noUploadFileRemote",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
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
			path:                "ignoreFolderLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: false,
		},
		checkedFileOrFolder{
			path:                "noDownloadFolderLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "noUploadFolderLocal",
			shouldExistInLocal:  true,
			shouldExistInRemote: false,
		},

		checkedFileOrFolder{
			path:                "testFolder3",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "ignoreFolderRemote",
			shouldExistInLocal:  true,
			shouldExistInRemote: false,
		},
		checkedFileOrFolder{
			path:                "noDownloadFolderRemote",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "noUploadFolderRemote",
			shouldExistInLocal:  true,
			shouldExistInRemote: false,
		},
	}

	checkFilesAndFolders(t, filesToCheck, foldersToCheck, local, remote)
}

func TestRunningSync(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on windows")
	}

	remote, local := initTestDirs(t)
	defer os.RemoveAll(remote)
	defer os.RemoveAll(local)

	syncClient := createTestSyncClient(local, remote)
	defer syncClient.Stop()

	syncClient.errorChan = make(chan error)

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

	// Start sync and do initial sync
	go syncClient.startUpstream()
	go syncClient.startDownstream()

	// Create
	err = createFileAndWait(remote, local, "2")
	if err != nil {
		t.Error(err)
		return
	}

	err = createFileAndWait(local, remote, "1")
	if err != nil {
		t.Error(err)
		return
	}

	err = createFolderAndWait(local, remote, "1")
	if err != nil {
		t.Error(err)
		return
	}

	err = createFolderAndWait(remote, local, "2")
	if err != nil {
		t.Error(err)
		return
	}

	// Remove
	err = removeFileAndWait(local, remote, "1")
	if err != nil {
		t.Error(err)
		return
	}

	err = removeFileAndWait(remote, local, "2")
	if err != nil {
		t.Error(err)
		return
	}

	err = removeFolderAndWait(local, remote, "1")
	if err != nil {
		t.Error(err)
		return
	}

	err = removeFolderAndWait(remote, local, "2")
	if err != nil {
		t.Error(err)
		return
	}

	// Check if there is an error in the error channel
	select {
	case err = <-syncClient.errorChan:
		t.Error(err)
		return
	default:
	}
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
