package sync

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/devspace-cloud/devspace/sync/util"

	"github.com/pkg/errors"
	gitignore "github.com/sabhiram/go-gitignore"
)

func untarAll(reader io.Reader, destPath, prefix string, config *Sync) error {
	fileCounter := 0
	gzr, err := gzip.NewReader(reader)
	if err != nil {
		return errors.Errorf("Error decompressing: %v", err)
	}

	defer gzr.Close()

	tarReader := tar.NewReader(gzr)
	for {
		shouldContinue, err := untarNext(tarReader, destPath, prefix, config)
		if err != nil {
			return errors.Wrap(err, "untarNext")
		} else if shouldContinue == false {
			return nil
		}

		fileCounter++
		if fileCounter%500 == 0 {
			config.log.Infof("Downstream - Untared %d files...", fileCounter)
		}
	}
}

func untarNext(tarReader *tar.Reader, destPath, prefix string, config *Sync) (bool, error) {
	config.fileIndex.fileMapMutex.Lock()
	defer config.fileIndex.fileMapMutex.Unlock()

	header, err := tarReader.Next()
	if err != nil {
		if err != io.EOF {
			return false, errors.Wrap(err, "tar next")
		}

		return false, nil
	}

	relativePath := getRelativeFromFullPath("/"+header.Name, prefix)
	outFileName := path.Join(destPath, relativePath)
	baseName := path.Dir(outFileName)

	// Check if newer file is there and then don't override?
	stat, err := os.Stat(outFileName)
	if err == nil {
		if stat.ModTime().Unix() > header.FileInfo().ModTime().Unix() {
			// Update filemap otherwise we download and download again
			config.fileIndex.fileMap[relativePath] = &FileInformation{
				Name:        relativePath,
				Mtime:       stat.ModTime().Unix(),
				Size:        stat.Size(),
				IsDirectory: stat.IsDir(),
			}

			if stat.IsDir() == false {
				config.log.Infof("Downstream - Don't override %s because file has newer mTime timestamp", relativePath)
			}
			return true, nil
		}
	}

	if err := os.MkdirAll(baseName, 0755); err != nil {
		return false, errors.Wrap(err, "mkdir all base")
	}

	if header.FileInfo().IsDir() {
		if err := os.MkdirAll(outFileName, 0755); err != nil {
			return false, errors.Wrap(err, "mkdir all")
		}

		config.fileIndex.CreateDirInFileMap(relativePath)
		return true, nil
	}

	// Create base dir in file map if it not already exists
	config.fileIndex.CreateDirInFileMap(getRelativeFromFullPath(baseName, destPath))

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
	}

	// Set mod time correctly
	_ = os.Chtimes(outFileName, time.Now(), header.ModTime)

	// Update fileMap so that upstream does not upload the file
	config.fileIndex.fileMap[relativePath] = &FileInformation{
		Name:        relativePath,
		Mtime:       header.ModTime.Unix(),
		Size:        header.FileInfo().Size(),
		IsDirectory: false,
	}

	return true, nil
}

// RecursiveTar runs recursively over the given path and basepath and tars the found files and folders
func RecursiveTar(basePath, relativePath string, writtenFiles map[string]*FileInformation, tw *tar.Writer, ignoreMatcher gitignore.IgnoreParser) error {
	if writtenFiles == nil {
		writtenFiles = make(map[string]*FileInformation)
	}

	absFilepath := path.Join(basePath, relativePath)
	if writtenFiles[relativePath] != nil {
		return nil
	}

	// We skip files that are suddenly not there anymore
	stat, err := os.Stat(absFilepath)
	if err != nil {
		// config.Logf("[Upstream] Couldn't stat file %s: %s\n", absFilepath, err.Error())
		return nil
	}

	// Exclude files on the exclude list
	if ignoreMatcher != nil && util.MatchesPath(ignoreMatcher, relativePath, stat.IsDir()) {
		return nil
	}

	fileInformation := createFileInformationFromStat(relativePath, stat)
	if stat.IsDir() {
		// Recursively tar folder
		return tarFolder(basePath, fileInformation, writtenFiles, stat, tw, ignoreMatcher)
	}

	return tarFile(basePath, fileInformation, writtenFiles, stat, tw)
}

func tarFolder(basePath string, fileInformation *FileInformation, writtenFiles map[string]*FileInformation, stat os.FileInfo, tw *tar.Writer, ignoreMatcher gitignore.IgnoreParser) error {
	filepath := path.Join(basePath, fileInformation.Name)
	files, err := ioutil.ReadDir(filepath)
	if err != nil {
		// config.Logf("[Upstream] Couldn't read dir %s: %s\n", filepath, err.Error())
		return nil
	}

	if len(files) == 0 && fileInformation.Name != "" {
		// Case empty directory
		hdr, _ := tar.FileInfoHeader(stat, filepath)
		hdr.Name = fileInformation.Name
		if err := tw.WriteHeader(hdr); err != nil {
			return errors.Wrap(err, "tar write header")
		}

		writtenFiles[fileInformation.Name] = fileInformation
	}

	for _, f := range files {
		if err := RecursiveTar(basePath, path.Join(fileInformation.Name, f.Name()), writtenFiles, tw, ignoreMatcher); err != nil {
			return errors.Wrap(err, "recursive tar "+f.Name())
		}
	}

	return nil
}

func tarFile(basePath string, fileInformation *FileInformation, writtenFiles map[string]*FileInformation, stat os.FileInfo, tw *tar.Writer) error {
	var err error
	filepath := path.Join(basePath, fileInformation.Name)
	if stat.Mode()&os.ModeSymlink == os.ModeSymlink {
		if filepath, err = os.Readlink(filepath); err != nil {
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

	hdr, err := tar.FileInfoHeader(stat, filepath)
	if err != nil {
		return errors.Wrap(err, "create tar file info header")
	}
	hdr.Name = fileInformation.Name
	hdr.ModTime = time.Unix(fileInformation.Mtime, 0)

	if err := tw.WriteHeader(hdr); err != nil {
		return errors.Wrap(err, "tar write header")
	}

	// nothing more to do for non-regular
	if !stat.Mode().IsRegular() {
		return nil
	}

	if copied, err := io.CopyN(tw, f, stat.Size()); err != nil {
		return errors.Wrap(err, "tar copy file")
	} else if copied != stat.Size() {
		return errors.New("tar: file truncated during read")
	}

	writtenFiles[fileInformation.Name] = fileInformation
	return nil
}

func createFileInformationFromStat(relativePath string, stat os.FileInfo) *FileInformation {
	return &FileInformation{
		Name:        relativePath,
		Size:        stat.Size(),
		Mtime:       stat.ModTime().Unix(),
		MtimeNano:   stat.ModTime().UnixNano(),
		IsDirectory: stat.IsDir(),
	}
}
