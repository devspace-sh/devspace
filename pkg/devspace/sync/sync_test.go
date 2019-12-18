package sync

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/sync/server"
	"github.com/pkg/errors"
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

	testRemotePath, err = filepath.EvalSymlinks(testRemotePath)
	if err != nil {
		t.Fatal(err)
	}

	testLocalPath, err = filepath.EvalSymlinks(testLocalPath)
	if err != nil {
		t.Fatal(err)
	}

	outside, err = filepath.EvalSymlinks(outside)
	if err != nil {
		t.Fatal(err)
	}

	return testRemotePath, testLocalPath, outside
}

func createTestSyncClient(testLocalPath string, testCases testCaseList) (*Sync, error) {
	syncLog = log.GetInstance()

	sync, err := NewSync(testLocalPath, getSyncOptions(testCases))
	if err != nil {
		return nil, err
	}

	sync.Options.SyncError = make(chan error)
	return sync, nil
}

func TestInitialSync(t *testing.T) {
	for _, downloadOnInitialSync := range []bool{true, false} {
		t.Log("DownloadOnInitialSync: " + strconv.FormatBool(downloadOnInitialSync))
		remote, local, outside := initTestDirs(t)
		defer os.RemoveAll(remote)
		defer os.RemoveAll(local)
		defer os.RemoveAll(outside)

		filesToCheck, foldersToCheck := makeBasicTestCases()
		if !downloadOnInitialSync {
			filesToCheck = disableDownload(filesToCheck)
			foldersToCheck = disableDownload(foldersToCheck)
		}

		// Start the client
		syncClient, err := createTestSyncClient(local, append(filesToCheck, foldersToCheck...))
		if err != nil {
			t.Fatal(err)
		}
		defer syncClient.Stop(nil)
		syncClient.Options.DownloadOnInitialSync = downloadOnInitialSync

		// Set bandwidth limits
		syncClient.Options.DownstreamLimit = 1024
		syncClient.Options.UpstreamLimit = 512

		// Start the downstream server
		downClientReader, downClientWriter, _ := os.Pipe()
		downServerReader, downServerWriter, _ := os.Pipe()
		defer downClientReader.Close()
		defer downClientWriter.Close()
		defer downServerReader.Close()
		defer downServerWriter.Close()

		// Build exclude paths
		excludePaths := []string{}
		excludePaths = append(excludePaths, syncClient.Options.ExcludePaths...)
		excludePaths = append(excludePaths, syncClient.Options.DownloadExcludePaths...)

		go func() {
			err := server.StartDownstreamServer(remote, excludePaths, downServerReader, downClientWriter, false)
			if err != nil {
				t.Fatal(err)
			}
		}()

		// Start downstream client
		err = syncClient.InitDownstream(downClientReader, downServerWriter)
		if err != nil {
			t.Fatal(err)
		}

		// Start upstream server
		upClientReader, upClientWriter, _ := os.Pipe()
		upServerReader, upServerWriter, _ := os.Pipe()
		defer upClientReader.Close()
		defer upClientWriter.Close()
		defer upServerReader.Close()
		defer upServerWriter.Close()

		go func() {
			err := server.StartUpstreamServer(remote, []string{}, upServerReader, upClientWriter, false)
			if err != nil {
				t.Fatal(err)
			}
		}()

		// Start upstream client
		err = syncClient.InitUpstream(upClientReader, upServerWriter)
		if err != nil {
			t.Fatal(err)
		}

		// Create test landscape
		err = createTestFilesAndFolders(local, remote, outside, filesToCheck, foldersToCheck)
		if err != nil {
			t.Fatal(err)
		}

		go syncClient.startUpstream()

		// Do initial sync
		err = syncClient.initialSync()
		if err != nil {
			t.Fatal(err)
		}

		checkFilesAndFolders(t, filesToCheck, foldersToCheck, local, remote, 15*time.Second)
	}
}

func TestNormalSync(t *testing.T) {
	remote, local, outside := initTestDirs(t)
	defer os.RemoveAll(remote)
	defer os.RemoveAll(local)
	defer os.RemoveAll(outside)

	filesToCheck, foldersToCheck := makeBasicTestCases()
	filesToCheck, foldersToCheck = makeRemoveAndRenameTestCases(filesToCheck, foldersToCheck)
	sort.Stable(foldersToCheck)

	syncClient, err := createTestSyncClient(local, append(filesToCheck, foldersToCheck...))
	if err != nil {
		t.Fatal(err)
	}
	defer syncClient.Stop(nil)

	// Start the downstream server
	downClientReader, downClientWriter, _ := os.Pipe()
	downServerReader, downServerWriter, _ := os.Pipe()
	defer downClientReader.Close()
	defer downClientWriter.Close()
	defer downServerReader.Close()
	defer downServerWriter.Close()

	// Build exclude paths
	excludePaths := []string{}
	excludePaths = append(excludePaths, syncClient.Options.ExcludePaths...)
	excludePaths = append(excludePaths, syncClient.Options.DownloadExcludePaths...)

	go func() {
		err := server.StartDownstreamServer(remote, excludePaths, downServerReader, downClientWriter, false)
		if err != nil {
			t.Fatal(err)
		}
	}()

	// Start downstream client
	err = syncClient.InitDownstream(downClientReader, downServerWriter)
	if err != nil {
		t.Fatal(err)
	}

	// Start upstream server
	upClientReader, upClientWriter, _ := os.Pipe()
	upServerReader, upServerWriter, _ := os.Pipe()
	defer upClientReader.Close()
	defer upClientWriter.Close()
	defer upServerReader.Close()
	defer upServerWriter.Close()

	go func() {
		err := server.StartUpstreamServer(remote, []string{}, upServerReader, upClientWriter, false)
		if err != nil {
			t.Fatal(err)
		}
	}()

	// Start upstream client
	err = syncClient.InitUpstream(upClientReader, upServerWriter)
	if err != nil {
		t.Fatal(err)
	}

	syncClient.readyChan = make(chan bool)

	go syncClient.startUpstream()
	go syncClient.startDownstream()

	<-syncClient.readyChan

	t.Log("Sync is ready")

	err = createTestFilesAndFolders(local, remote, outside, filesToCheck, foldersToCheck)
	if err != nil {
		t.Error(err)
		return
	}
	checkFilesAndFolders(t, filesToCheck, foldersToCheck, local, remote, 15*time.Second)

	t.Log("Create test is done")

	filesToCheck, foldersToCheck, err = removeSomeTestFilesAndFolders(local, remote, filesToCheck, foldersToCheck, "_Remove")
	if err != nil {
		t.Error(err)
		return
	}
	checkFilesAndFolders(t, filesToCheck, foldersToCheck, local, remote, 15*time.Second)

	t.Log("Delete test is done")

	filesToCheck, foldersToCheck, err = renameSomeTestFilesAndFolders(local, remote, outside, filesToCheck, foldersToCheck)
	if err != nil {
		t.Error(err)
		return
	}
	checkFilesAndFolders(t, filesToCheck, foldersToCheck, local, remote, 15*time.Second)

	t.Log("Rename test is done")

	// @Florian TODO: Test upstream symlinks
}

func getSyncOptions(testCases testCaseList) *Options {
	options := &Options{
		ExcludePaths:         []string{},
		DownloadExcludePaths: []string{},
		UploadExcludePaths:   []string{},
		Verbose:              true,
	}

	for _, testCase := range testCases {
		/*
			All paths that should be in these ExcludePaths are marked like this with these strings.
			for Example: ignoreFileLocal
			The RenameTo... parts of some files contain those, too, but those use Big Letters so they are not excluded.
			For example: testFileLocal_RenameToIgnore
		*/
		if strings.Contains(testCase.path, "ignore") {
			options.ExcludePaths = append(options.ExcludePaths, testCase.path)
		} else if strings.Contains(testCase.path, "noDownload") {
			options.DownloadExcludePaths = append(options.DownloadExcludePaths, testCase.path)
		} else if strings.Contains(testCase.path, "noUpload") {
			options.UploadExcludePaths = append(options.UploadExcludePaths, testCase.path)
		} else if strings.HasSuffix(testCase.path, "_RenameToIgnore") {
			options.ExcludePaths = append(options.ExcludePaths, testCase.path+"After")
		} else if strings.HasSuffix(testCase.path, "_RenameToNoDownload") {
			options.DownloadExcludePaths = append(options.DownloadExcludePaths, testCase.path+"After")
		} else if strings.HasSuffix(testCase.path, "_RenameToNoUpload") {
			options.UploadExcludePaths = append(options.UploadExcludePaths, testCase.path+"After")
		}
	}

	return options
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

	filesToCheck = makeSymLinkTestCases(filesToCheck)
	foldersToCheck = makeSymLinkTestCases(foldersToCheck)

	//Add Files and Folders that are inside a shared testFolder
	filesToCheck = makeDeepTestCases(filesToCheck)
	foldersToCheck = makeDeepTestCases(foldersToCheck)

	return filesToCheck, foldersToCheck
}

func disableDownload(testCases testCaseList) testCaseList {
	for i, f := range testCases {
		if !strings.Contains(f.path, "Remote") {
			continue
		}
		testCases[i].shouldExistInLocal = false
		testCases[i].shouldExistInRemote = strings.Contains(f.path, "ignore") || strings.Contains(f.path, "noDownload")
	}
	return testCases
}

func makeRemoveAndRenameTestCases(filesToCheck testCaseList, foldersToCheck testCaseList) (testCaseList, testCaseList) {
	for n, array := range []testCaseList{filesToCheck, foldersToCheck} {
		for _, f := range array {
			if f.path == "testFolder" {
				continue
			}

			removeEquivalent := checkedFileOrFolder{
				path:                f.path + "_Remove",
				shouldExistInLocal:  f.shouldExistInLocal,
				shouldExistInRemote: f.shouldExistInRemote,
				editLocation:        f.editLocation,
				isSymLink:           f.isSymLink,
			}
			array = append(array, removeEquivalent)

			renameEquivalent := checkedFileOrFolder{
				path:                f.path + "_RenameToFullContext",
				shouldExistInLocal:  f.shouldExistInLocal,
				shouldExistInRemote: f.shouldExistInRemote,
				editLocation:        f.editLocation,
				isSymLink:           f.isSymLink,
			}
			array = append(array, renameEquivalent)

			isFullyIncluded, _ := regexp.Compile("(testFolder\\/)?(testFile|testFolder)(Local|Remote)$")
			if isFullyIncluded.MatchString(f.path) {
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToOutside",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
					editLocation:        f.editLocation,
					isSymLink:           f.isSymLink,
				}
				array = append(array, renameEquivalent)
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToIgnore",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
					editLocation:        f.editLocation,
					isSymLink:           f.isSymLink,
				}
				array = append(array, renameEquivalent)
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToNoDownload",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
					editLocation:        f.editLocation,
					isSymLink:           f.isSymLink,
				}
				array = append(array, renameEquivalent)
				renameEquivalent = checkedFileOrFolder{
					path:                f.path + "_RenameToNoUpload",
					shouldExistInLocal:  f.shouldExistInLocal,
					shouldExistInRemote: f.shouldExistInRemote,
					editLocation:        f.editLocation,
					isSymLink:           f.isSymLink,
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

	renameFilesFromOutside = makeSymLinkTestCases(renameFilesFromOutside)
	renameFolderFromOutside = makeSymLinkTestCases(renameFolderFromOutside)

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
			path:                filepath.Join("testFolder", f.path),
			shouldExistInLocal:  f.shouldExistInLocal,
			shouldExistInRemote: f.shouldExistInRemote,
			editLocation:        f.editLocation,
		}
		testCases = append(testCases, deepEquivalent)
	}

	return testCases
}

func makeSymLinkTestCases(testCases testCaseList) testCaseList {
	for _, f := range testCases {
		if f.isSymLink || strings.Contains(f.path, "Remote") || f.path == "testFolder" {
			continue
		}

		deepEquivalent := checkedFileOrFolder{
			path:                strings.ReplaceAll(strings.ReplaceAll(f.path, "File", "SymLinkToFile"), "Folder", "SymLinkToFolder"),
			shouldExistInLocal:  f.shouldExistInLocal,
			shouldExistInRemote: f.shouldExistInRemote,
			editLocation:        f.editLocation,
			isSymLink:           true,
		}
		testCases = append(testCases, deepEquivalent)
	}

	return testCases
}

func createTestFilesAndFolders(local string, remote string, outside string, filesToCheck testCaseList, foldersToCheck testCaseList) error {
	for _, f := range foldersToCheck {
		createLocation := f.editLocation
		if f.isSymLink {
			createLocation = editSymLinkDir
		}

		parentDir, err := getParentDir(local, remote, outside, createLocation)
		if err != nil {
			return errors.Wrap(err, "get parent dir")
		}

		err = os.MkdirAll(path.Join(parentDir, f.path), 0755)
		if err != nil {
			return errors.Wrap(err, "make dir")
		}

		if f.isSymLink {
			symLinkParentDir, err := getParentDir(local, remote, outside, f.editLocation)
			if err != nil {
				return errors.Wrap(err, "get parent dir for symLink")
			}
			err = os.Symlink(path.Join(parentDir, f.path), path.Join(symLinkParentDir, f.path))
			if err != nil {
				return errors.Wrap(err, "make symLink")
			}
		}
	}

	for _, f := range filesToCheck {
		createLocation := f.editLocation
		if f.isSymLink {
			createLocation = editSymLinkDir
		}
		parentDir, err := getParentDir(local, remote, outside, createLocation)
		if err != nil {
			return errors.Wrap(err, "get parent dir from "+f.path)
		}

		err = ioutil.WriteFile(path.Join(parentDir, f.path), []byte(fileContents), 0666)
		if err != nil {
			return errors.Wrap(err, "write file")
		}

		if f.isSymLink {
			symLinkParentDir, err := getParentDir(local, remote, outside, f.editLocation)
			if err != nil {
				return errors.Wrap(err, "get parent dir for symLink")
			}
			err = os.Symlink(path.Join(parentDir, f.path), path.Join(symLinkParentDir, f.path))
			if err != nil {
				return errors.Wrap(err, "make symLink")
			}
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
	for n, array := range []testCaseList{filesToCheck, foldersToCheck} {
		for n, f := range array {
			if !strings.Contains(f.path, "_Rename") {
				continue
			}

			fromParentDir, err := getParentDir(local, remote, outside, f.editLocation)
			if err != nil {
				return nil, nil, errors.Wrap(err, "get parent dir")
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
				return nil, nil, errors.Wrap(err, "rename")
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
	sync := Sync{
		fileIndex: newFileIndex(),
	}

	sync.fileIndex.CreateDirInFileMap("/TestDir1/TestDir2/TestDir3/TestDir4")

	if len(sync.fileIndex.fileMap) != 4 {
		t.Fatal("Create dir in file map failed!")
	}
}

func TestRemoveDirInFileMap(t *testing.T) {
	sync := Sync{
		fileIndex: newFileIndex(),
	}

	sync.fileIndex.fileMap = map[string]*FileInformation{
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
		t.Fatal("Remove dir in file map failed!")
	}
}
