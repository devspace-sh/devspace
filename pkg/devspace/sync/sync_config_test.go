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

	filesToCheck, foldersToCheck := createTestFilesAndFolders(local, remote)

	go syncClient.startUpstream()

	// Do initial sync
	err = syncClient.initialSync()
	if err != nil {
		t.Error(err)
		return
	}

	checkFilesAndFolders(t, filesToCheck, foldersToCheck, local, remote, 10*time.Second)
}
func TestNormalSync(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on windows")
	}

	remote, local := initTestDirs(t)
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

	syncClient.readyChan = make(chan bool)

	go syncClient.startUpstream()
	go syncClient.startDownstream()

	<-syncClient.readyChan

	filesToCheck, foldersToCheck := createTestFilesAndFolders(local, remote)
	if err != nil {
		t.Error(err)
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
		"ignoreFileLocal_Remove",
		"ignoreFolderLocal_Remove",
		"testFolder/ignoreFileLocal_Remove",
		"ignoreFileRemote_Remove",
		"ignoreFolderRemote_Remove",
		"testFolder/ignoreFileRemote_Remove",
	}

	syncClient.DownloadExcludePaths = []string{
		"noDownloadFileLocal",
		"noDownloadFolderLocal",
		"testFolder/noDownloadFileLocal",
		"noDownloadFileRemote",
		"noDownloadFolderRemote",
		"testFolder/noDownloadFileRemote",
		"noDownloadFileLocal_Remove",
		"noDownloadFolderLocal_Remove",
		"testFolder/noDownloadFileLocal_Remove",
		"noDownloadFileRemote_Remove",
		"noDownloadFolderRemote_Remove",
		"testFolder/noDownloadFileRemote_Remove",
	}

	syncClient.UploadExcludePaths = []string{
		"noUploadFileLocal",
		"noUploadFolderLocal",
		"testFolder/noUploadFileLocal",
		"noUploadFileRemote",
		"noUploadFolderRemote",
		"testFolder/noUploadFileRemote",
		"noUploadFileLocal_Remove",
		"noUploadFolderLocal_Remove",
		"testFolder/noUploadFileLocal_Remove",
		"noUploadFileRemote_Remove",
		"noUploadFolderRemote_Remove",
		"testFolder/noUploadFileRemote_Remove",
	}

	syncClient.initIgnoreParsers()

}

func createTestFilesAndFolders(local string, remote string) ([]checkedFileOrFolder, []checkedFileOrFolder) {

	//Write local files
	ioutil.WriteFile(path.Join(local, "testFileLocal1"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFileLocal2"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "ignoreFileLocal"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "noDownloadFileLocal"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "noUploadFileLocal"), []byte(fileContents), 0666)

	os.Mkdir(path.Join(local, "testFolder"), 0755)
	os.Mkdir(path.Join(local, "testFolderLocal"), 0755)
	os.Mkdir(path.Join(local, "ignoreFolderLocal"), 0755)
	os.Mkdir(path.Join(local, "noDownloadFolderLocal"), 0755)
	os.Mkdir(path.Join(local, "noUploadFolderLocal"), 0755)

	ioutil.WriteFile(path.Join(local, "testFolder", "testFileLocal1"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFolder", "testFileLocal2"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFolder", "ignoreFileLocal"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFolder", "noDownloadFileLocal"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFolder", "noUploadFileLocal"), []byte(fileContents), 0666)

	// Write remote files
	ioutil.WriteFile(path.Join(remote, "testFileRemote1"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "testFileRemote2"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "ignoreFileRemote"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "noDownloadFileRemote"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "noUploadFileRemote"), []byte(fileContents), 0666)

	os.Mkdir(path.Join(remote, "testFolder"), 0755)
	os.Mkdir(path.Join(remote, "testFolderRemote"), 0755)
	os.Mkdir(path.Join(remote, "ignoreFolderRemote"), 0755)
	os.Mkdir(path.Join(remote, "noDownloadFolderRemote"), 0755)
	os.Mkdir(path.Join(remote, "noUploadFolderRemote"), 0755)

	ioutil.WriteFile(path.Join(remote, "testFolder", "testFileRemote1"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "testFolder", "testFileRemote2"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "testFolder", "ignoreFileRemote"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "testFolder", "noDownloadFileRemote"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "testFolder", "noUploadFileRemote"), []byte(fileContents), 0666)

	//-----------The following files will be removed later-------------------------------------
	//Write local files
	ioutil.WriteFile(path.Join(local, "testFileLocal1_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFileLocal2_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "ignoreFileLocal_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "noDownloadFileLocal_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "noUploadFileLocal_Remove"), []byte(fileContents), 0666)

	os.Mkdir(path.Join(local, "testFolder_Remove"), 0755)
	os.Mkdir(path.Join(local, "testFolderLocal_Remove"), 0755)
	os.Mkdir(path.Join(local, "ignoreFolderLocal_Remove"), 0755)
	os.Mkdir(path.Join(local, "noDownloadFolderLocal_Remove"), 0755)
	os.Mkdir(path.Join(local, "noUploadFolderLocal_Remove"), 0755)

	ioutil.WriteFile(path.Join(local, "testFolder", "testFileLocal1_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFolder", "testFileLocal2_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFolder", "ignoreFileLocal_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFolder", "noDownloadFileLocal_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFolder", "noUploadFileLocal_Remove"), []byte(fileContents), 0666)

	// Write remote files
	ioutil.WriteFile(path.Join(remote, "testFileRemote1_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "testFileRemote2_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "ignoreFileRemote_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "noDownloadFileRemote_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "noUploadFileRemote_Remove"), []byte(fileContents), 0666)

	os.Mkdir(path.Join(remote, "testFolder_Remove"), 0755)
	os.Mkdir(path.Join(remote, "testFolderRemote_Remove"), 0755)
	os.Mkdir(path.Join(remote, "ignoreFolderRemote_Remove"), 0755)
	os.Mkdir(path.Join(remote, "noDownloadFolderRemote_Remove"), 0755)
	os.Mkdir(path.Join(remote, "noUploadFolderRemote_Remove"), 0755)

	ioutil.WriteFile(path.Join(remote, "testFolder", "testFileRemote1_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "testFolder", "testFileRemote2_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "testFolder", "ignoreFileRemote_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "testFolder", "noDownloadFileRemote_Remove"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "testFolder", "noUploadFileRemote_Remove"), []byte(fileContents), 0666)

	filesToCheck := []checkedFileOrFolder{
		checkedFileOrFolder{
			path:                "testFileLocal1",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "testFileLocal2",
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
			path:                "testFolder/testFileLocal1",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "testFolder/testFileLocal2",
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
			path:                "testFileRemote1",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "testFileRemote2",
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
			path:                "testFolder/testFileRemote1",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "testFolder/testFileRemote2",
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
			path:                "testFolderLocal",
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
			path:                "testFolderRemote",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "ignoreFolderRemote",
			shouldExistInLocal:  false,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "noDownloadFolderRemote",
			shouldExistInLocal:  false,
			shouldExistInRemote: true,
		},
		checkedFileOrFolder{
			path:                "noUploadFolderRemote",
			shouldExistInLocal:  true,
			shouldExistInRemote: true,
		},
	}

	for _, f := range filesToCheck {
		removeEquivalent := checkedFileOrFolder{
			path:                f.path + "_Remove",
			shouldExistInLocal:  f.shouldExistInLocal,
			shouldExistInRemote: f.shouldExistInRemote,
		}
		filesToCheck = append(filesToCheck, removeEquivalent)
	}
	for _, f := range foldersToCheck {
		removeEquivalent := checkedFileOrFolder{
			path:                f.path + "_Remove",
			shouldExistInLocal:  f.shouldExistInLocal,
			shouldExistInRemote: f.shouldExistInRemote,
		}
		foldersToCheck = append(foldersToCheck, removeEquivalent)
	}

	return filesToCheck, foldersToCheck
}

func removeSomeTestFilesAndFolders(local string, remote string, filesToCheck []checkedFileOrFolder, foldersToCheck []checkedFileOrFolder, removeSuffix string) ([]checkedFileOrFolder, []checkedFileOrFolder, error) {

	removeIfSuffixMatch := func(path string, f os.FileInfo, err error) error {
		if strings.HasSuffix(path, removeSuffix) {
			return os.RemoveAll(path)
		}
		return nil
	}

	err := filepath.Walk(remote, removeIfSuffixMatch)
	if err != nil {
		return nil, nil, err
	}
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
