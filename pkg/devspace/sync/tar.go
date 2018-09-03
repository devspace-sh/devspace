package sync

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/juju/errors"
)

func untarAll(reader io.Reader, destPath, prefix string, config *SyncConfig) error {
	entrySeq := 0

	// TODO: use compression here?
	tarReader := tar.NewReader(reader)

	for {
		shouldContinue, err := untarNext(tarReader, entrySeq, destPath, prefix, config)

		if err != nil {
			return errors.Trace(err)
		} else if shouldContinue == false {
			return nil
		}

		entrySeq++

		if entrySeq%500 == 0 {
			config.Logf("[Downstream] Untared %d files...\n", entrySeq)
		}
	}
}

func untarNext(tarReader *tar.Reader, entrySeq int, destPath, prefix string, config *SyncConfig) (bool, error) {
	config.fileIndex.fileMapMutex.Lock()
	defer config.fileIndex.fileMapMutex.Unlock()

	header, err := tarReader.Next()
	if err != nil {
		if err != io.EOF {
			return false, errors.Trace(err)
		}

		return false, nil
	}

	mode := header.FileInfo().Mode()
	relativePath := clean(header.Name[len(prefix):])
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

	config.fileIndex.CreateDirInFileMap(getRelativeFromFullPath(baseName, destPath))

	// handle coping remote file into local directory
	if entrySeq == 0 && !header.FileInfo().IsDir() {
		exists, err := dirExists(outFileName)
		if err != nil {
			return false, errors.Trace(err)
		}
		if exists {
			outFileName = filepath.Join(outFileName, path.Base(clean(header.Name)))
		}
	}

	if mode&os.ModeSymlink != 0 {
		// Skip the processing of symlinks for now, because windows has problems with them
		// err := os.Symlink(header.Linkname, outFileName)
		// if err != nil {
		//	 return errors.Trace(err)
		// }
	} else {
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

		err = os.Chtimes(outFileName, time.Now(), header.FileInfo().ModTime())

		if err != nil {
			return false, errors.Trace(err)
		}

		relativePath = getRelativeFromFullPath(outFileName, destPath)

		// Update fileMap so that upstream does not upload the file
		config.fileIndex.fileMap[relativePath] = &fileInformation{
			Name:        relativePath,
			Mtime:       header.FileInfo().ModTime().Unix(),
			Size:        header.FileInfo().Size(),
			IsDirectory: false,
		}
	}

	return true, nil
}

func writeTar(files []*fileInformation, config *SyncConfig) (string, map[string]*fileInformation, error) {
	f, err := ioutil.TempFile("", "")

	if err != nil {
		return "", nil, errors.Trace(err)
	}

	defer f.Close()

	tarWriter := tar.NewWriter(f)
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
