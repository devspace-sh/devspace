package sync

import (
	"archive/tar"
	"compress/gzip"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/loft-sh/devspace/helper/server/ignoreparser"

	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/pkg/errors"
)

// Unarchiver is responsible for unarchiving a remote archive
type Unarchiver struct {
	syncConfig    *Sync
	forceOverride bool

	log log.Logger
}

// NewUnarchiver creates a new unarchiver
func NewUnarchiver(syncConfig *Sync, forceOverride bool, log log.Logger) *Unarchiver {
	return &Unarchiver{
		syncConfig:    syncConfig,
		forceOverride: forceOverride,
		log:           log,
	}
}

// Untar untars the given reader into the destination directory
func (u *Unarchiver) Untar(fromReader io.ReadCloser, toPath string) error {
	defer fromReader.Close()

	fileCounter := 0
	gzr, err := gzip.NewReader(fromReader)
	if err != nil {
		return errors.Errorf("error decompressing: %v", err)
	}

	defer gzr.Close()

	tarReader := tar.NewReader(gzr)
	for {
		shouldContinue, err := u.untarNext(toPath, tarReader)
		if err != nil {
			return errors.Wrapf(err, "decompress %s", toPath)
		} else if !shouldContinue {
			return nil
		}

		fileCounter++
		if fileCounter%500 == 0 {
			u.log.Infof("Downstream - Untared %d files...", fileCounter)
		}
	}
}

func (u *Unarchiver) untarNext(destPath string, tarReader *tar.Reader) (bool, error) {
	u.syncConfig.fileIndex.fileMapMutex.Lock()
	defer u.syncConfig.fileIndex.fileMapMutex.Unlock()

	header, err := tarReader.Next()
	if err != nil {
		if err != io.EOF {
			return false, errors.Wrap(err, "tar next")
		}

		return false, nil
	}

	relativePath := getRelativeFromFullPath("/"+header.Name, "")
	outFileName := path.Join(destPath, relativePath)
	baseName := path.Dir(outFileName)

	// Check if newer file is there and then don't override?
	stat, err := os.Stat(outFileName)
	if err == nil && !u.forceOverride {
		if stat.ModTime().Unix() > header.FileInfo().ModTime().Unix() {
			// Update filemap otherwise we download and download again
			u.syncConfig.fileIndex.fileMap[relativePath] = &FileInformation{
				Name:        relativePath,
				Mtime:       stat.ModTime().Unix(),
				Mode:        stat.Mode(),
				Size:        stat.Size(),
				IsDirectory: stat.IsDir(),
			}

			if !stat.IsDir() {
				u.syncConfig.log.Infof("Downstream - Don't override %s because file has newer mTime timestamp", relativePath)
			}
			return true, nil
		}
	}

	if err := u.createAllFolders(baseName, 0755); err != nil {
		return false, err
	}

	if header.FileInfo().IsDir() {
		if err := u.createAllFolders(outFileName, 0755); err != nil {
			return false, err
		}

		u.syncConfig.fileIndex.CreateDirInFileMap(relativePath)
		return true, nil
	}

	// Create base dir in file map if it not already exists
	u.syncConfig.fileIndex.CreateDirInFileMap(getRelativeFromFullPath(baseName, destPath))

	// Create / Override file
	outFile, err := os.Create(outFileName)
	if err != nil {
		// Try again after 5 seconds
		time.Sleep(time.Second * 5)
		outFile, err = os.Create(outFileName)
		if err != nil {
			return false, errors.Wrap(err, "create file")
		}
	}

	defer outFile.Close()
	if _, err := io.Copy(outFile, tarReader); err != nil {
		return false, errors.Wrap(err, "copy file to reader")
	}

	if err := outFile.Close(); err != nil {
		return false, errors.Wrap(err, "close file")
	}

	if stat != nil {
		// Set old permissions correctly
		_ = os.Chmod(outFileName, stat.Mode())

		// Set owner & group correctly
		// TODO: Enable this on supported platforms
		// _ = os.Chown(outFileName, stat.Sys().(*syscall.Stat).Uid, stat.Sys().(*syscall.Stat_t).Gid)
	} else {
		// Set permissions
		_ = os.Chmod(outFileName, header.FileInfo().Mode())
	}

	// Set mod time correctly
	_ = os.Chtimes(outFileName, time.Now(), header.ModTime)

	// Update fileMap so that upstream does not upload the file
	u.syncConfig.fileIndex.fileMap[relativePath] = &FileInformation{
		Name:        relativePath,
		Mtime:       header.ModTime.Unix(),
		Mode:        header.FileInfo().Mode(),
		Size:        header.FileInfo().Size(),
		IsDirectory: false,
	}

	return true, nil
}

func (u *Unarchiver) createAllFolders(name string, perm os.FileMode) error {
	absPath, err := filepath.Abs(name)
	if err != nil {
		return err
	}

	slashPath := filepath.ToSlash(absPath)
	pathParts := strings.Split(slashPath, "/")
	for i := 1; i < len(pathParts); i++ {
		dirToCreate := strings.Join(pathParts[:i+1], "/")
		err := os.Mkdir(dirToCreate, perm)
		if err != nil {
			if os.IsExist(err) {
				continue
			}

			return errors.Errorf("Error creating %s: %v", dirToCreate, err)
		}
	}

	return nil
}

// Archiver is responsible for compressing specific files and folders within a target directory
type Archiver struct {
	basePath      string
	ignoreMatcher ignoreparser.IgnoreParser
	writer        *tar.Writer
	writtenFiles  map[string]*FileInformation
}

// NewArchiver creates a new archiver
func NewArchiver(basePath string, writer *tar.Writer, ignoreMatcher ignoreparser.IgnoreParser) *Archiver {
	return &Archiver{
		basePath: basePath,

		ignoreMatcher: ignoreMatcher,
		writer:        writer,
		writtenFiles:  make(map[string]*FileInformation),
	}
}

// WrittenFiles returns the written files by the archiver
func (a *Archiver) WrittenFiles() map[string]*FileInformation {
	return a.writtenFiles
}

// AddToArchive adds a new path to the archive
func (a *Archiver) AddToArchive(relativePath string) error {
	absFilepath := path.Join(a.basePath, relativePath)
	if a.writtenFiles[relativePath] != nil {
		return nil
	}

	// We skip files that are suddenly not there anymore
	stat, err := os.Stat(absFilepath)
	if err != nil {
		// config.Logf("[Upstream] Couldn't stat file %s: %s\n", absFilepath, err.Error())
		return nil
	}

	// Exclude files on the exclude list if it does not have a negate pattern, otherwise we will check below
	if a.ignoreMatcher != nil && !a.ignoreMatcher.RequireFullScan() && a.ignoreMatcher.Matches(relativePath, stat.IsDir()) {
		return nil
	}

	fileInformation := createFileInformationFromStat(relativePath, stat)
	if stat.IsDir() {
		// Recursively tar folder
		return a.tarFolder(fileInformation, stat)
	}

	// exclude file?
	if a.ignoreMatcher == nil || !a.ignoreMatcher.RequireFullScan() || !a.ignoreMatcher.Matches(relativePath, false) {
		return a.tarFile(fileInformation, stat)
	}

	return nil
}

func (a *Archiver) tarFolder(target *FileInformation, targetStat os.FileInfo) error {
	filePath := path.Join(a.basePath, target.Name)
	files, err := os.ReadDir(filePath)
	if err != nil {
		// config.Logf("[Upstream] Couldn't read dir %s: %s\n", filepath, err.Error())
		return nil
	}

	if len(files) == 0 && target.Name != "" {
		// check if not excluded
		if a.ignoreMatcher == nil || !a.ignoreMatcher.RequireFullScan() || !a.ignoreMatcher.Matches(target.Name, true) {
			// Case empty directory
			hdr, _ := tar.FileInfoHeader(targetStat, filePath)
			hdr.Uid = 0
			hdr.Gid = 0
			hdr.Mode = fillGo18FileTypeBits(int64(chmodTarEntry(os.FileMode(hdr.Mode))), targetStat)
			hdr.Name = target.Name
			if err := a.writer.WriteHeader(hdr); err != nil {
				return errors.Wrap(err, "tar write header")
			}

			a.writtenFiles[target.Name] = target
		}
	}

	for _, dirEntry := range files {
		f, err := dirEntry.Info()
		if err != nil {
			continue
		}

		if fsutil.IsRecursiveSymlink(f, path.Join(filePath, f.Name())) {
			continue
		}

		if err = a.AddToArchive(path.Join(target.Name, f.Name())); err != nil {
			return errors.Wrap(err, "recursive tar "+f.Name())
		}
	}

	return nil
}

func (a *Archiver) tarFile(target *FileInformation, targetStat os.FileInfo) error {
	var err error
	filepath := path.Join(a.basePath, target.Name)
	if targetStat.Mode()&os.ModeSymlink == os.ModeSymlink {
		if filepath, err = os.Readlink(filepath); err != nil {
			return nil
		}

		targetStat, err = os.Stat(filepath)
		if err != nil || targetStat.IsDir() {
			// We ignore open file and just treat it as okay
			return nil
		}
	}

	// Case regular file
	f, err := os.Open(filepath)
	if err != nil {
		// We ignore open file and just treat it as okay
		return nil
	}
	defer f.Close()

	hdr, err := tar.FileInfoHeader(targetStat, filepath)
	if err != nil {
		return errors.Wrap(err, "create tar file info header")
	}
	hdr.Name = target.Name
	hdr.Uid = 0
	hdr.Gid = 0
	hdr.Mode = fillGo18FileTypeBits(int64(chmodTarEntry(os.FileMode(hdr.Mode))), targetStat)
	hdr.ModTime = time.Unix(target.Mtime, 0)

	if err := a.writer.WriteHeader(hdr); err != nil {
		return errors.Wrap(err, "tar write header")
	}

	// nothing more to do for non-regular
	if !targetStat.Mode().IsRegular() {
		return nil
	}

	copied, err := io.CopyN(a.writer, f, targetStat.Size())
	if err != nil {
		return errors.Wrap(err, "tar copy file")
	} else if copied != targetStat.Size() {
		return errors.New("tar: file truncated during read")
	}

	a.writtenFiles[target.Name] = target
	return nil
}

const (
	modeISDIR  = 040000  // Directory
	modeISFIFO = 010000  // FIFO
	modeISREG  = 0100000 // Regular file
	modeISLNK  = 0120000 // Symbolic link
	modeISBLK  = 060000  // Block special file
	modeISCHR  = 020000  // Character special file
	modeISSOCK = 0140000 // Socket
)

// chmodTarEntry is used to adjust the file permissions used in tar header based
// on the platform the archival is done.
func chmodTarEntry(perm os.FileMode) os.FileMode {
	if runtime.GOOS != "windows" {
		return perm
	}

	// perm &= 0755 // this 0-ed out tar flags (like link, regular file, directory marker etc.)
	permPart := perm & os.ModePerm
	noPermPart := perm &^ os.ModePerm
	// Add the x bit: make everything +x from windows
	permPart |= 0111
	permPart &= 0755

	return noPermPart | permPart
}

// fillGo18FileTypeBits fills type bits which have been removed on Go 1.9 archive/tar
// https://github.com/golang/go/commit/66b5a2f
func fillGo18FileTypeBits(mode int64, fi os.FileInfo) int64 {
	fm := fi.Mode()
	switch {
	case fm.IsRegular():
		mode |= modeISREG
	case fi.IsDir():
		mode |= modeISDIR
	case fm&os.ModeSymlink != 0:
		mode |= modeISLNK
	case fm&os.ModeDevice != 0:
		if fm&os.ModeCharDevice != 0 {
			mode |= modeISCHR
		} else {
			mode |= modeISBLK
		}
	case fm&os.ModeNamedPipe != 0:
		mode |= modeISFIFO
	case fm&os.ModeSocket != 0:
		mode |= modeISSOCK
	}
	return mode
}

func createFileInformationFromStat(relativePath string, stat os.FileInfo) *FileInformation {
	return &FileInformation{
		Name:        relativePath,
		Size:        stat.Size(),
		Mtime:       stat.ModTime().Unix(),
		MtimeNano:   stat.ModTime().UnixNano(),
		Mode:        stat.Mode(),
		IsDirectory: stat.IsDir(),
	}
}
