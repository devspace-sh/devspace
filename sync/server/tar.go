package server

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type fileInformation struct {
	Name  string
	Size  int64
	Mtime time.Time
}

func untarAll(reader io.Reader, destPath, prefix string) error {
	gzr, err := gzip.NewReader(reader)
	if err != nil {
		return errors.Errorf("Error decompressing: %v", err)
	}

	defer gzr.Close()
	tarReader := tar.NewReader(gzr)

	for {
		shouldContinue, err := untarNext(tarReader, destPath, prefix)
		if err != nil {
			return errors.Wrap(err, "untarNext")
		} else if shouldContinue == false {
			return nil
		}
	}
}

func untarNext(tarReader *tar.Reader, destPath, prefix string) (bool, error) {
	header, err := tarReader.Next()
	if err != nil {
		if err != io.EOF {
			return false, errors.Wrap(err, "tar reader next")
		}

		return false, nil
	}

	relativePath := getRelativeFromFullPath("/"+header.Name, prefix)
	outFileName := path.Join(destPath, relativePath)
	baseName := path.Dir(outFileName)

	// Check if newer file is there and then don't override?
	stat, _ := os.Stat(outFileName)

	if err := os.MkdirAll(baseName, 0755); err != nil {
		return false, errors.Wrap(err, "mkdir all "+baseName)
	}

	if header.FileInfo().IsDir() {
		if err := os.MkdirAll(outFileName, 0755); err != nil {
			return false, errors.Wrap(err, "mkdir all "+outFileName)
		}

		return true, nil
	}

	// Create / Override file
	outFile, err := os.Create(outFileName)
	if err != nil {
		// Try again after 5 seconds
		time.Sleep(time.Second * 5)
		outFile, err = os.Create(outFileName)
		if err != nil {
			return false, errors.Wrap(err, "create "+outFileName)
		}
	}

	defer outFile.Close()

	if _, err := io.Copy(outFile, tarReader); err != nil {
		return false, errors.Wrap(err, "io copy tar reader")
	}
	if err := outFile.Close(); err != nil {
		return false, errors.Wrap(err, "out file close")
	}

	// Set old permissions and owner and group
	if stat != nil {
		// Set old permissions correctly
		_ = os.Chmod(outFileName, stat.Mode())

		// Set old owner & group correctly
		_ = Chown(outFileName, stat)
	}

	// Set mod time from tar header
	_ = os.Chtimes(outFileName, time.Now(), header.FileInfo().ModTime())

	return true, nil
}

func recursiveTar(basePath, relativePath string, writtenFiles map[string]bool, tw *tar.Writer, skipFolderContents bool) error {
	absFilepath := path.Join(basePath, relativePath)
	if _, ok := writtenFiles[relativePath]; ok {
		return nil
	}

	// We skip files that are suddenly not there anymore
	stat, err := os.Stat(absFilepath)
	if err != nil {
		// File is suddenly not here anymore is ignored
		return nil
	}

	fileInformation := createFileInformationFromStat(relativePath, stat)
	if stat.IsDir() {
		// Recursively tar folder
		return tarFolder(basePath, fileInformation, writtenFiles, stat, tw, skipFolderContents)
	}

	return tarFile(basePath, fileInformation, writtenFiles, stat, tw)
}

func tarFolder(basePath string, fileInformation *fileInformation, writtenFiles map[string]bool, stat os.FileInfo, tw *tar.Writer, skipContents bool) error {
	filepath := path.Join(basePath, fileInformation.Name)
	files, err := ioutil.ReadDir(filepath)
	if err != nil {
		// Ignore this error because it could happen the file is suddenly not there anymore
		return nil
	}

	if skipContents || (len(files) == 0 && fileInformation.Name != "") {
		// Case empty directory
		hdr, _ := tar.FileInfoHeader(stat, filepath)
		hdr.Name = fileInformation.Name
		hdr.ModTime = fileInformation.Mtime
		if err := tw.WriteHeader(hdr); err != nil {
			return errors.Wrapf(err, "tw write header %s", filepath)
		}

		writtenFiles[fileInformation.Name] = true
	}

	if skipContents == false {
		for _, f := range files {
			if err := recursiveTar(basePath, path.Join(fileInformation.Name, f.Name()), writtenFiles, tw, skipContents); err != nil {
				return errors.Wrap(err, "recursive tar")
			}
		}
	}

	return nil
}

func tarFile(basePath string, fileInformation *fileInformation, writtenFiles map[string]bool, stat os.FileInfo, tw *tar.Writer) error {
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
		// We ignore this error here because it could happen that the file is suddenly not here anymore
		return nil
	}
	defer f.Close()

	hdr, err := tar.FileInfoHeader(stat, filepath)
	if err != nil {
		return errors.Wrapf(err, "tar file info header %s", filepath)
	}

	hdr.Name = fileInformation.Name

	// We have to cut of the nanoseconds otherwise this sometimes leads to issues on the client side
	// because the unix value will be rounded up
	hdr.ModTime = time.Unix(fileInformation.Mtime.Unix(), 0)

	if err := tw.WriteHeader(hdr); err != nil {
		return errors.Wrapf(err, "tw write header %s", filepath)
	}

	// nothing more to do for non-regular
	if !stat.Mode().IsRegular() {
		return nil
	}

	if copied, err := io.CopyN(tw, f, stat.Size()); err != nil {
		return errors.Wrapf(err, "io copy %s", filepath)
	} else if copied != stat.Size() {
		return errors.New("tar: file truncated during read")
	}

	writtenFiles[fileInformation.Name] = true
	return nil
}

func getRelativeFromFullPath(fullpath string, prefix string) string {
	return strings.TrimPrefix(strings.Replace(strings.Replace(fullpath[len(prefix):], "\\", "/", -1), "//", "/", -1), ".")
}

func createFileInformationFromStat(relativePath string, stat os.FileInfo) *fileInformation {
	fileInformation := &fileInformation{
		Name:  relativePath,
		Size:  stat.Size(),
		Mtime: stat.ModTime(),
	}

	return fileInformation
}
