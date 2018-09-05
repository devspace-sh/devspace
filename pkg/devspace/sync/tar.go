package sync

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/juju/errors"
)

func untarAll(reader io.Reader, destPath, prefix string, config *SyncConfig) error {
	fileCounter := 0
	gzr, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("Error decompressing: %v", err)
	}

	defer gzr.Close()

	tarReader := tar.NewReader(gzr)

	for {
		shouldContinue, err := untarNext(tarReader, destPath, prefix, config)

		if err != nil {
			return errors.Trace(err)
		} else if shouldContinue == false {
			return nil
		}

		fileCounter++

		if fileCounter%500 == 0 {
			config.Logf("[Downstream] Untared %d files...\n", fileCounter)
		}
	}
}

func untarNext(tarReader *tar.Reader, destPath, prefix string, config *SyncConfig) (bool, error) {
	config.fileIndex.fileMapMutex.Lock()
	defer config.fileIndex.fileMapMutex.Unlock()

	header, err := tarReader.Next()
	if err != nil {
		if err != io.EOF {
			return false, errors.Trace(err)
		}

		return false, nil
	}

	relativePath := getRelativeFromFullPath("/"+header.Name, prefix)
	outFileName := path.Join(destPath, relativePath)
	baseName := path.Dir(outFileName)

	// Check if newer file is there and then don't override?
	stat, err := os.Stat(outFileName)

	if err == nil {
		if ceilMtime(stat.ModTime()) > header.FileInfo().ModTime().Unix() {
			// Update filemap otherwise we download and download again
			config.fileIndex.fileMap[relativePath] = &fileInformation{
				Name:        relativePath,
				Mtime:       ceilMtime(stat.ModTime()),
				Size:        stat.Size(),
				IsDirectory: stat.IsDir(),
			}

			config.Logf("[Downstream] Don't override %s because file has newer mTime timestamp\n", relativePath)
			return true, nil
		}
	}

	if err := os.MkdirAll(baseName, 0755); err != nil {
		return false, errors.Trace(err)
	}

	if header.FileInfo().IsDir() {
		if err := os.MkdirAll(outFileName, 0755); err != nil {
			return false, errors.Trace(err)
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
			return false, errors.Trace(err)
		}
	}

	defer outFile.Close()

	if _, err := io.Copy(outFile, tarReader); err != nil {
		return false, errors.Trace(err)
	}

	if err := outFile.Close(); err != nil {
		return false, errors.Trace(err)
	}

	if stat != nil {
		// Set old permissions correctly
		_ = os.Chmod(outFileName, stat.Mode())

		// Set owner & group correctly
		// TODO: Enable this on supported platforms
		// _ = os.Chown(outFileName, stat.Sys().(*syscall.Stat).Uid, stat.Sys().(*syscall.Stat_t).Gid)
	}

	// Set mod time correctly
	err = os.Chtimes(outFileName, time.Now(), header.FileInfo().ModTime())

	if err != nil {
		return false, errors.Trace(err)
	}

	// Update fileMap so that upstream does not upload the file
	config.fileIndex.fileMap[relativePath] = &fileInformation{
		Name:        relativePath,
		Mtime:       header.FileInfo().ModTime().Unix(),
		Size:        header.FileInfo().Size(),
		IsDirectory: false,
	}

	return true, nil
}

func writeTar(files []*fileInformation, config *SyncConfig) (string, map[string]*fileInformation, error) {
	f, err := ioutil.TempFile("", "")

	if err != nil {
		return "", nil, errors.Trace(err)
	}

	defer f.Close()

	// Use compression
	gw := gzip.NewWriter(f)
	defer gw.Close()

	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	writtenFiles := make(map[string]*fileInformation)

	for _, element := range files {
		relativePath := element.Name

		if writtenFiles[relativePath] == nil {
			err := recursiveTar(config.WatchPath, relativePath, "", relativePath, writtenFiles, tarWriter, config)

			if err != nil {
				config.Logf("[Upstream] Tar failed: %s. Will retry in 4 seconds...\n", err.Error())
				os.Remove(f.Name())

				time.Sleep(time.Second * 4)

				return writeTar(files, config)
			}
		}
	}

	return f.Name(), writtenFiles, nil
}

// TODO: Error handling if files are not there
func recursiveTar(srcBase, srcFile, destBase, destFile string, writtenFiles map[string]*fileInformation, tw *tar.Writer, config *SyncConfig) error {
	filepath := path.Join(srcBase, srcFile)
	relativePath := getRelativeFromFullPath(filepath, srcBase)

	if writtenFiles[relativePath] != nil {
		return nil
	}

	// Exclude files on the exclude list
	if config.ignoreMatcher != nil {
		if config.ignoreMatcher.MatchesPath(relativePath) {
			return nil
		}
	}

	// Exclude files on the upload exclude list
	if config.uploadIgnoreMatcher != nil {
		if config.uploadIgnoreMatcher.MatchesPath(relativePath) {
			return nil
		}
	}

	stat, err := os.Lstat(filepath)

	// We skip files that are suddenly not there anymore
	if err != nil {
		config.Logf("[Upstream] Couldn't stat file %s: %s\n", filepath, err.Error())

		return nil
	}

	// We skip symlinks
	if stat.Mode()&os.ModeSymlink != 0 {
		return nil
	}

	fileInformation := &fileInformation{
		Name:        relativePath,
		Size:        stat.Size(),
		Mtime:       ceilMtime(stat.ModTime()),
		IsDirectory: stat.IsDir(),
	}

	config.fileIndex.fileMapMutex.Lock()
	if config.fileIndex.fileMap[relativePath] != nil {
		fileInformation.RemoteMode = config.fileIndex.fileMap[relativePath].RemoteMode
		fileInformation.RemoteGID = config.fileIndex.fileMap[relativePath].RemoteGID
		fileInformation.RemoteUID = config.fileIndex.fileMap[relativePath].RemoteUID
	}
	config.fileIndex.fileMapMutex.Unlock()

	if stat.IsDir() {
		files, err := ioutil.ReadDir(filepath)

		if err != nil {
			config.Logf("[Upstream] Couldn't read dir %s: %s\n", filepath, err.Error())
			return nil
		}

		if len(files) == 0 && relativePath != "" {
			//case empty directory
			hdr, _ := tar.FileInfoHeader(stat, filepath)
			hdr.Name = strings.Replace(destFile, "\\", "/", -1) // Need to replace \ with / for windows

			config.fileIndex.fileMapMutex.Lock()
			if config.fileIndex.fileMap[relativePath] != nil {
				hdr.Mode = fileInformation.RemoteMode
				hdr.Uid = fileInformation.RemoteUID
				hdr.Gid = fileInformation.RemoteGID
			}
			config.fileIndex.fileMapMutex.Unlock()

			if err := tw.WriteHeader(hdr); err != nil {
				return errors.Trace(err)
			}

			writtenFiles[relativePath] = fileInformation
		}

		for _, f := range files {
			if err := recursiveTar(srcBase, path.Join(srcFile, f.Name()), destBase, path.Join(destFile, f.Name()), writtenFiles, tw, config); err != nil {
				return errors.Trace(err)
			}
		}

		return nil
	}

	f, err := os.Open(filepath)

	if err != nil {
		return errors.Trace(err)
	}

	defer f.Close()

	//case regular file or other file type like pipe
	hdr, err := tar.FileInfoHeader(stat, filepath)

	if err != nil {
		return errors.Trace(err)
	}

	hdr.Name = strings.Replace(destFile, "\\", "/", -1)

	config.fileIndex.fileMapMutex.Lock()
	if config.fileIndex.fileMap[relativePath] != nil {
		hdr.Mode = fileInformation.RemoteMode
		hdr.Uid = fileInformation.RemoteUID
		hdr.Gid = fileInformation.RemoteGID
	}
	config.fileIndex.fileMapMutex.Unlock()

	if err := tw.WriteHeader(hdr); err != nil {
		return errors.Trace(err)
	}

	if _, err := io.Copy(tw, f); err != nil {
		return errors.Trace(err)
	}

	writtenFiles[relativePath] = fileInformation

	return f.Close()
}
