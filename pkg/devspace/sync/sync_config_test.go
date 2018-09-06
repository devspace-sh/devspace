package sync

import (
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/covexo/devspace/pkg/util/log"
)

var (
	testRemotePath string
	testLocalPath  string
	syncClient     *SyncConfig
)

func createTestSyncClient(t *testing.T) {
	var err error

	if syncClient == nil {
		testRemotePath, err = ioutil.TempDir("", "")
		if err != nil {
			t.Fatalf("Couldn't create test dir: %v", err)
		}

		testLocalPath, err = ioutil.TempDir("", "")
		if err != nil {
			t.Fatalf("Couldn't create test dir: %v", err)
		}

		destPath := testRemotePath

		if runtime.GOOS == "windows" {
			destPath = strings.Replace(destPath, "\\", "/", -1)
			destPath = "/mnt/" + strings.ToLower(string(destPath[0])) + destPath[2:]
		}

		// Log to stdout
		syncLog = log.GetInstance()
		syncClient = &SyncConfig{
			WatchPath: strings.Replace(testLocalPath, "\\", "/", -1),
			DestPath:  destPath,

			testing: true,
		}

		// Start client
		err = syncClient.Start()

		if err != nil {
			t.Fatalf("Couldn't init test sync client: %v", err)
		}
	}
}

func cleanupTestClient() {
	if syncClient != nil {
		syncClient.Stop()

		os.RemoveAll(testLocalPath)
		os.RemoveAll(testRemotePath)

		syncClient = nil
	}
}

func createFileLocal(t *testing.T) {
	filename := path.Join(testLocalPath, "testFile1")
	filenameRemote := path.Join(testRemotePath, "testFile1")
	fileContents := "testFile1"

	ioutil.WriteFile(filename, []byte(fileContents), 0666)

	for i := 0; i < 100; i++ {
		if _, err := os.Stat(filenameRemote); err == nil {
			data, err := ioutil.ReadFile(filenameRemote)

			if err != nil {
				t.Fatalf("Created file %s could not be read: %v", filenameRemote, err)
			}

			if string(data) != fileContents {
				t.Fatalf("Created file %s wrong contents: got %s, expected %s", filenameRemote, string(data), fileContents)
			}

			return
		}

		time.Sleep(time.Millisecond * 100)
	}

	t.Fatalf("Created file %s wasn't synced to %s", filename, filenameRemote)
}

func TestSyncEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}

	createTestSyncClient(t)
	createFileLocal(t)
	cleanupTestClient()
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
