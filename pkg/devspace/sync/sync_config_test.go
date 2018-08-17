package sync

import (
	"testing"
	"time"
)

func TestCreateDirInFileMap(t *testing.T) {
	sync := SyncConfig{
		fileMap: map[string]*FileInformation{},
	}

	sync.createDirInFileMap("/TestDir1/TestDir2/TestDir3/TestDir4")

	if len(sync.fileMap) != 4 {
		t.Error("Create dir in file map failed!")
		t.Fail()
	}
}
func TestRemoveDirInFileMap(t *testing.T) {
	sync := SyncConfig{
		fileMap: map[string]*FileInformation{
			"/TestDir": &FileInformation{
				Name:        "/TestDir",
				IsDirectory: true,
			},
			"/TestDir/File1": &FileInformation{
				Name:        "/TestDir/File1",
				Size:        1234,
				Mtime:       1234,
				IsDirectory: false,
			},
			"/TestDir2": &FileInformation{
				Name:        "/TestDir2",
				IsDirectory: true,
			},
		},
	}

	sync.removeDirInFileMap("/TestDir")

	if len(sync.fileMap) != 1 {
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

/* func TestSync(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}

	done := make(chan bool)

	sync := SyncConfig{
		//PodName:   "node-75c5b7bbbd-8gcpz",
		WatchPath: "D:\\Programmieren\\go-workspace\\src\\git.covexo.com\\covexo\\devspace\\.test",
		DestPath:  "/home",
		ExcludeRegEx: []string{
			"Test.txt",
		},
	}

	sync.Start()

	<-done
} */
