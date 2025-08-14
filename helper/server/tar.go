package server

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/loft-sh/devspace/pkg/util/fsutil"

	"github.com/pkg/errors"
)

type fileInformation struct {
	Name  string
	Size  int64
	Mtime time.Time
}

func untarAll(reader io.ReadCloser, options *UpstreamOptions) error {
	defer reader.Close()

	gzr, err := gzip.NewReader(reader)
	if err != nil {
		return errors.Errorf("error decompressing: %v", err)
	}
	defer gzr.Close()

	tarReader := tar.NewReader(gzr)
	for {
		shouldContinue, err := untarNext(tarReader, options)
		if err != nil {
			return errors.Wrap(err, "decompress")
		} else if !shouldContinue {
			return nil
		}
	}
}

func createAllFolders(name string, perm os.FileMode, options *UpstreamOptions) error {
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

			return errors.Errorf("error creating %s: %v", dirToCreate, err)
		}

		if options.DirCreateCmd != "" {
			cmdArgs := make([]string, 0, len(options.DirCreateArgs))
			for _, arg := range options.DirCreateArgs {
				if arg == "{}" {
					cmdArgs = append(cmdArgs, dirToCreate)
				} else {
					cmdArgs = append(cmdArgs, arg)
				}
			}

			out, err := exec.Command(options.DirCreateCmd, cmdArgs...).CombinedOutput()
			if err != nil {
				return errors.Errorf("error executing command '%s %s': %s => %v", options.DirCreateCmd, strings.Join(cmdArgs, " "), string(out), err)
			}
		}
	}

	return nil
}

func untarNext(tarReader *tar.Reader, options *UpstreamOptions) (bool, error) {
	header, err := tarReader.Next()
	if err != nil {
		if err != io.EOF {
			return false, errors.Wrap(err, "tar reader next")
		}

		return false, nil
	}

	relativePath := getRelativeFromFullPath("/"+header.Name, "")
	outFileName := path.Join(options.UploadPath, relativePath)
	baseName := path.Dir(outFileName)

	// Check if newer file is there and then don't override?
	stat, _ := os.Stat(outFileName)

	if err := createAllFolders(baseName, 0755, options); err != nil {
		return false, err
	}

	if header.FileInfo().IsDir() {
		if err := createAllFolders(outFileName, 0755, options); err != nil {
			return false, err
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
			return false, errors.Wrapf(err, "create %s", outFileName)
		}
	}

	defer outFile.Close()

	if _, err := io.Copy(outFile, tarReader); err != nil {
		return false, errors.Wrapf(err, "io copy tar reader %s", outFileName)
	}
	if err := outFile.Close(); err != nil {
		return false, errors.Wrapf(err, "out file close %s", outFileName)
	}

	// Set old permissions and owner and group
	if stat != nil {
		if options.OverridePermission {
			// Set permissions
			_ = os.Chmod(outFileName, header.FileInfo().Mode())
		} else {
			// Set old permissions correctly
			_ = os.Chmod(outFileName, stat.Mode())
		}

		// Set old owner & group correctly
		_ = Chown(outFileName, stat)
	} else {
		// Set permissions
		_ = os.Chmod(outFileName, header.FileInfo().Mode())
	}

	// Set mod time from tar header
	_ = os.Chtimes(outFileName, time.Now(), header.FileInfo().ModTime())

	// Execute command if defined
	if options.FileChangeCmd != "" {
		cmdArgs := make([]string, 0, len(options.FileChangeArgs))
		for _, arg := range options.FileChangeArgs {
			if arg == "{}" {
				cmdArgs = append(cmdArgs, outFileName)
			} else {
				cmdArgs = append(cmdArgs, arg)
			}
		}

		out, err := exec.Command(options.FileChangeCmd, cmdArgs...).CombinedOutput()
		if err != nil {
			return false, errors.Errorf("error executing command '%s %s': %s => %v", options.FileChangeCmd, strings.Join(cmdArgs, " "), string(out), err)
		}
	}

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
	files, err := os.ReadDir(filepath)
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

	if !skipContents {
		for _, dirEntry := range files {
			f, err := dirEntry.Info()
			if err != nil {
				continue
			}

			if fsutil.IsRecursiveSymlink(f, path.Join(fileInformation.Name, f.Name())) {
				continue
			}

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
	return strings.TrimPrefix(strings.ReplaceAll(strings.ReplaceAll(fullpath[len(prefix):], "\\", "/"), "//", "/"), ".")
}

func createFileInformationFromStat(relativePath string, stat os.FileInfo) *fileInformation {
	fileInformation := &fileInformation{
		Name:  relativePath,
		Size:  stat.Size(),
		Mtime: stat.ModTime(),
	}

	return fileInformation
}
