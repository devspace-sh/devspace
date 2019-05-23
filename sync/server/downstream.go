package server

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/devspace-cloud/devspace/sync/remote"
	"github.com/devspace-cloud/devspace/sync/util"
	"github.com/pkg/errors"
	gitignore "github.com/sabhiram/go-gitignore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// StartDownstreamServer starts a new downstream server with the given reader and writer
func StartDownstreamServer(remotePath string, reader io.Reader, writer io.Writer) error {
	pipe := util.NewStdStreamJoint(reader, writer)
	lis := util.NewStdinListener()
	done := make(chan error)

	go func() {
		s := grpc.NewServer()

		remote.RegisterDownstreamServer(s, &Downstream{
			RemotePath: remotePath,
		})
		reflection.Register(s)

		done <- s.Serve(lis)
	}()

	lis.Ready(pipe)
	return <-done
}

// Downstream is the implementation for the downstream server
type Downstream struct {
	// RemotePath is the path to watch for changes
	RemotePath string

	// watchedFiles is a memory map of the previous state of the changes function
	watchedFiles map[string]*watchedFile
}

type watchedFile struct {
	Name  string
	Size  int64
	Mtime int64

	IsDir bool
}

// Download sends the file at the temp download location to the client
func (d *Downstream) Download(stream remote.Downstream_DownloadServer) error {
	reader, writer, err := os.Pipe()
	if err != nil {
		return errors.Wrap(err, "create pipe")
	}

	defer reader.Close()
	defer writer.Close()

	err = d.compress(writer, stream)
	if err != nil {
		return errors.Wrap(err, "compress paths")
	}

	writer.Close()

	buf := make([]byte, 16*1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			err := stream.Send(&remote.Chunk{
				Content: buf[:n],
			})
			if err != nil {
				return errors.Wrap(err, "stream send")
			}
		}

		if err == io.EOF {
			return nil
		} else if err != nil {
			return errors.Wrap(err, "read file")
		}
	}
}

// Compress compresses the given files into a tar archive
func (d *Downstream) compress(writer io.Writer, stream remote.Downstream_DownloadServer) error {
	// Use compression
	gw := gzip.NewWriter(writer)
	defer gw.Close()

	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	writtenFiles := make(map[string]*fileInformation)
	for {
		path, err := stream.Recv()
		if err == io.EOF {
			return nil
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

// Changes retrieves all changes from the watchpath
func (d *Downstream) Changes(excluded *remote.Excluded, stream remote.Downstream_ChangesServer) error {
	ignoreMatcher, err := compilePaths(excluded.Paths)
	if err != nil {
		return errors.Wrap(err, "compile paths")
	}

	newState := make(map[string]*watchedFile)

	// Walk through the dir
	walkDir(d.RemotePath, ignoreMatcher, newState)

	err = streamChanges(d.RemotePath, d.watchedFiles, newState, stream)
	if err != nil {
		return nil
	}

	d.watchedFiles = newState
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
					Path:       newFile.Name[len(basePath):],
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
				Path:       newFile.Name[len(basePath):],
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
				Path:       oldFile.Name[len(basePath):],
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
			continue
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
