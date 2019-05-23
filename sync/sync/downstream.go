package sync

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/devspace-cloud/devspace/sync/remote"
	"github.com/pkg/errors"
	gitignore "github.com/sabhiram/go-gitignore"
)

// Downstream is the implementation for the downstream server
type Downstream struct {
	// RemotePath is the path to watch for changes
	RemotePath string

	// TempDownloadPath is the temporary download file path
	TempDownloadPath string

	// watchedFiles is a memory map of the previous state of the changes function
	watchedFiles map[string]*watchedFile
}

type watchedFile struct {
	Name  string
	Size  int64
	Mtime int64

	IsDir bool
}

// Compress compresses the given files into a tar archive
func (d *Downstream) Compress(stream remote.Downstream_CompressServer) error {
	f, err := os.OpenFile(d.TempDownloadPath, os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	// Use compression
	gw := gzip.NewWriter(f)
	defer gw.Close()

	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	writtenFiles := make(map[string]*fileInformation)
	for {
		path, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&remote.Empty{})
		}
		if err != nil {
			return err
		}

		if writtenFiles[path.Path] == nil {
			err := recursiveTar(d.RemotePath, path.Path, writtenFiles, tarWriter)
			if err != nil {
				return errors.Wrap(err, "recursive tar")
			}
		}
	}
}

// Download sends the file at the temp download location to the client
func (d *Downstream) Download(empty *remote.Empty, stream remote.Downstream_DownloadServer) error {
	f, err := os.Open(d.TempDownloadPath)
	if err != nil {
		return errors.Wrap(err, "open download file")
	}
	defer f.Close()

	buf := make([]byte, 16*1024)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			stream.Send(&remote.Chunk{
				Content: buf[:n],
			})
		}

		if err == io.EOF {
			return nil
		} else if err != nil {
			return errors.Wrap(err, "read file")
		}
	}
}

// Changes retrieves all changes from the
func (d *Downstream) Changes(excluded *remote.Excluded, stream remote.Downstream_ChangesServer) error {
	ignoreMatcher, err := compilePaths(excluded.Paths)
	if err != nil {
		return errors.Wrap(err, "compile paths")
	}

	newState := make(map[string]*watchedFile)

	// Walk through the dir
	walkDir(d.RemotePath, ignoreMatcher, newState)

	return nil
}

func streamChanges(basePath string, oldState map[string]*watchedFile, newState map[string]*watchedFile, stream remote.Downstream_ChangesServer) error {
	if oldState == nil {
		oldState = make(map[string]*watchedFile)
	}

	// Compare new -> old
	for _, newFile := range newState {
		if oldFile, ok := oldState[newFile.Name]; ok {
			if oldFile.IsDir != newFile.IsDir || oldFile.Size != newFile.Size || oldFile.Mtime != newFile.Mtime {
				err := stream.Send(&remote.Change{
					ChangeType: remote.ChangeType_CHANGE,
					Path:       newFile.Name[len(basePath)-1:],
					Mtime:      newFile.Mtime,
					Size:       newFile.Size,
					IsDir:      newFile.IsDir,
				})
				if err != nil {
					return err
				}
			}
		} else {
			err := stream.Send(&remote.Change{
				ChangeType: remote.ChangeType_CHANGE,
				Path:       newFile.Name[len(basePath)-1:],
				Mtime:      newFile.Mtime,
				Size:       newFile.Size,
				IsDir:      newFile.IsDir,
			})
			if err != nil {
				return err
			}
		}
	}

	// Compare old -> new
	for _, oldFile := range oldState {
		if _, ok := newState[oldFile.Name]; ok == false {
			err := stream.Send(&remote.Change{
				ChangeType: remote.ChangeType_DELETE,
				Path:       oldFile.Name[len(basePath)-1:],
				Mtime:      oldFile.Mtime,
				Size:       oldFile.Size,
				IsDir:      oldFile.IsDir,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func walkDir(path string, ignoreMatcher gitignore.IgnoreParser, state map[string]*watchedFile) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		// We ignore errors here
		return
	}

	for _, f := range files {
		absolutePath := filepath.Join(path, f.Name())
		if ignoreMatcher.MatchesPath(absolutePath) {
			return
		}

		if f.IsDir() {
			state[absolutePath] = &watchedFile{
				Name:  absolutePath,
				IsDir: true,
			}

			walkDir(absolutePath, ignoreMatcher, state)
		} else {
			state[absolutePath] = &watchedFile{
				Name:  absolutePath,
				Size:  f.Size(),
				Mtime: f.ModTime().Unix(),
				IsDir: false,
			}
		}
	}
}

func compilePaths(excludePaths []string) (gitignore.IgnoreParser, error) {
	if len(excludePaths) > 0 {
		ignoreParser, err := gitignore.CompileIgnoreLines(excludePaths...)
		if err != nil {
			return nil, errors.Wrap(err, "compile ignore lines")
		}

		return ignoreParser, nil
	}

	return nil, nil
}
