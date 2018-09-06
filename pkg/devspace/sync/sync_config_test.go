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

func removeFolderAndWait(from, to, postfix string, t *testing.T) error {
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

func removeFileAndWait(from, to, postfix string, t *testing.T) error {
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

func createFolderAndWait(from, to, postfix string, t *testing.T) error {
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

func createFileAndWait(from, to, postfix string, t *testing.T) error {
	filenameFrom := path.Join(from, "testFile"+postfix)
	filenameTo := path.Join(to, "testFile"+postfix)
	fileContents := "testFile" + postfix

	ioutil.WriteFile(filenameFrom, []byte(fileContents), 0666)

	for i := 0; i < 50; i++ {
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

		time.Sleep(time.Millisecond * 100)
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

	os.Mkdir(path.Join(local, "testFolder"), 0755)
	os.Mkdir(path.Join(local, "testFolder2"), 0755)

	ioutil.WriteFile(path.Join(local, "testFolder", "testFile1"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(local, "testFolder", "testFile2"), []byte(fileContents), 0666)

	// Write remote files
	ioutil.WriteFile(path.Join(remote, "testFile3"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "testFile4"), []byte(fileContents), 0666)

	os.Mkdir(path.Join(remote, "testFolder"), 0755)
	os.Mkdir(path.Join(remote, "testFolder3"), 0755)

	ioutil.WriteFile(path.Join(remote, "testFolder", "testFile3"), []byte(fileContents), 0666)
	ioutil.WriteFile(path.Join(remote, "testFolder", "testFile4"), []byte(fileContents), 0666)

	// Do initial sync
	err = syncClient.initialSync()
	if err != nil {
		t.Error(err)
		return
	}

	// Check outcome
	filesToCheck := []string{
		"testFile1",
		"testFile2",
		"testFile3",
		"testFile4",
		"testFolder/testFile1",
		"testFolder/testFile2",
		"testFolder/testFile3",
		"testFolder/testFile4",
	}

	foldersToCheck := []string{
		"testFolder",
		"testFolder2",
		"testFolder3",
	}

	// Check files
	for _, v := range filesToCheck {
		localFile := path.Join(local, v)
		remoteFile := path.Join(remote, v)

		_, err = os.Stat(localFile)
		if err != nil {
			t.Error(err)
			return
		}

		_, err = os.Stat(remoteFile)
		if err != nil {
			t.Error(err)
			return
		}

		data, err := ioutil.ReadFile(localFile)
		if err != nil {
			t.Error(err)
			return
		}
		if string(data) != fileContents {
			t.Errorf("Wrong file contentsin file %s, got %s, expected %s", localFile, string(data), fileContents)
		}

		data, err = ioutil.ReadFile(remoteFile)
		if err != nil {
			t.Error(err)
			return
		}
		if string(data) != fileContents {
			t.Errorf("Wrong file contentsin file %s, got %s, expected %s", remoteFile, string(data), fileContents)
		}
	}

	// Check folders
	for _, v := range foldersToCheck {
		localFolder := path.Join(local, v)
		remoteFolder := path.Join(remote, v)

		stat, err := os.Stat(localFolder)
		if err != nil {
			t.Error(err)
			return
		}
		if stat.IsDir() == false {
			t.Errorf("Expected %s to be a dir", localFolder)
			return
		}

		stat, err = os.Stat(remoteFolder)
		if err != nil {
			t.Error(err)
			return
		}
		if stat.IsDir() == false {
			t.Errorf("Expected %s to be a dir", remoteFolder)
			return
		}
	}

	// Check if there is an error in the error channel
	select {
	case err = <-syncClient.errorChan:
		t.Error(err)
		return
	default:
	}
}

func TestRunningSync(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on windows")
	}

	remote, local := initTestDirs(t)
	defer os.RemoveAll(remote)
	defer os.RemoveAll(local)

	syncClient := createTestSyncClient(remote, local)
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

	// Start sync and do inital sync
	syncClient.mainLoop()

	// Create
	err = createFileAndWait(local, remote, "1", t)
	if err != nil {
		t.Error(err)
		return
	}

	err = createFileAndWait(remote, local, "2", t)
	if err != nil {
		t.Error(err)
		return
	}

	err = createFolderAndWait(local, remote, "1", t)
	if err != nil {
		t.Error(err)
		return
	}

	err = createFolderAndWait(remote, local, "2", t)
	if err != nil {
		t.Error(err)
		return
	}

	// Remove
	err = removeFileAndWait(local, remote, "1", t)
	if err != nil {
		t.Error(err)
		return
	}

	err = removeFileAndWait(remote, local, "2", t)
	if err != nil {
		t.Error(err)
		return
	}

	err = removeFolderAndWait(local, remote, "1", t)
	if err != nil {
		t.Error(err)
		return
	}

	err = removeFolderAndWait(remote, local, "2", t)
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
