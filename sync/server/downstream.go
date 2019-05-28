package server

import (
	"archive/tar"
	"compress/gzip"
	"context"
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
func StartDownstreamServer(remotePath string, excludePaths []string, reader io.Reader, writer io.Writer) error {
	pipe := util.NewStdStreamJoint(reader, writer)
	lis := util.NewStdinListener()
	done := make(chan error)

	// Compile ignore paths
	ignoreMatcher, err := compilePaths(excludePaths)
	if err != nil {
		return errors.Wrap(err, "compile paths")
	}

	go func() {
		s := grpc.NewServer()

		remote.RegisterDownstreamServer(s, &Downstream{
			RemotePath:    remotePath,
			ignoreMatcher: ignoreMatcher,
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

	// ignore matcher is the ignore matcher which matches against excluded files and paths
	ignoreMatcher gitignore.IgnoreParser

	// watchedFiles is a memory map of the previous state of the changes function
	watchedFiles map[string]*remote.Change
}

// Download sends the file at the temp download location to the client
func (d *Downstream) Download(stream remote.Downstream_DownloadServer) error {
	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		return errors.Wrap(err, "create temp file")
	}
	defer os.Remove(tempFile.Name())

	err = d.compress(tempFile, stream)
	if err != nil {
		return errors.Wrap(err, "compress paths")
	}

	tempFile.Close()
	tempFile, err = os.Open(tempFile.Name())
	if err != nil {
		return errors.Wrap(err, "open temp file")
	}

	buf := make([]byte, 16*1024)
	for {
		n, err := tempFile.Read(buf)
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

// Compress compresses the given files and folders into a tar archive
func (d *Downstream) compress(writer io.Writer, stream remote.Downstream_DownloadServer) error {
	// Use compression
	gw := gzip.NewWriter(writer)
	defer gw.Close()

	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	writtenFiles := make(map[string]*fileInformation)
	for {
		paths, err := stream.Recv()
		if paths != nil {
			for _, path := range paths.Paths {
				if _, ok := writtenFiles[path]; ok == false {
					err := recursiveTar(d.RemotePath, path, writtenFiles, tarWriter, true)
					if err != nil {
						return errors.Wrap(err, "recursive tar")
					}
				}
			}
		}

		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// ChangesCount returns the amount of changes on the remote side
func (d *Downstream) ChangesCount(context.Context, *remote.Empty) (*remote.ChangeAmount, error) {
	newState := make(map[string]*remote.Change)

	// Walk through the dir
	walkDir(d.RemotePath, d.ignoreMatcher, newState)

	changeAmount, err := streamChanges(d.RemotePath, d.watchedFiles, newState, nil)
	if err != nil {
		return nil, errors.Wrap(err, "count changes")
	}

	return &remote.ChangeAmount{
		Amount: changeAmount,
	}, nil
}

// Changes retrieves all changes from the watchpath
func (d *Downstream) Changes(empty *remote.Empty, stream remote.Downstream_ChangesServer) error {
	newState := make(map[string]*remote.Change)

	// Walk through the dir
	walkDir(d.RemotePath, d.ignoreMatcher, newState)

	_, err := streamChanges(d.RemotePath, d.watchedFiles, newState, stream)
	if err != nil {
		return errors.Wrap(err, "stream changes")
	}

	d.watchedFiles = newState
	return nil
}

func streamChanges(basePath string, oldState map[string]*remote.Change, newState map[string]*remote.Change, stream remote.Downstream_ChangesServer) (int64, error) {
	changeAmount := int64(0)
	if oldState == nil {
		oldState = make(map[string]*remote.Change)
	}

	// Compare new -> old
	changes := make([]*remote.Change, 0, 64)
	for _, newFile := range newState {
		if oldFile, ok := oldState[newFile.Path]; ok {
			if oldFile.IsDir != newFile.IsDir || oldFile.Size != newFile.Size || oldFile.MtimeUnix != newFile.MtimeUnix || oldFile.MtimeUnixNano != newFile.MtimeUnixNano {
				if stream != nil {
					changes = append(changes, &remote.Change{
						ChangeType:    remote.ChangeType_CHANGE,
						Path:          newFile.Path[len(basePath):],
						MtimeUnix:     newFile.MtimeUnix,
						MtimeUnixNano: newFile.MtimeUnixNano,
						Size:          newFile.Size,
						IsDir:         newFile.IsDir,
					})
				}

				changeAmount++
			}
		} else {
			if stream != nil {
				changes = append(changes, &remote.Change{
					ChangeType:    remote.ChangeType_CHANGE,
					Path:          newFile.Path[len(basePath):],
					MtimeUnix:     newFile.MtimeUnix,
					MtimeUnixNano: newFile.MtimeUnixNano,
					Size:          newFile.Size,
					IsDir:         newFile.IsDir,
				})
			}

			changeAmount++
		}

		if len(changes) >= 64 && stream != nil {
			err := stream.Send(&remote.ChangeChunk{Changes: changes})
			if err != nil {
				return 0, errors.Wrap(err, "send changes")
			}

			changes = make([]*remote.Change, 0, 64)
		}
	}

	// Compare old -> new
	for _, oldFile := range oldState {
		if _, ok := newState[oldFile.Path]; ok == false {
			if stream != nil {
				changes = append(changes, &remote.Change{
					ChangeType:    remote.ChangeType_DELETE,
					Path:          oldFile.Path[len(basePath):],
					MtimeUnix:     oldFile.MtimeUnix,
					MtimeUnixNano: oldFile.MtimeUnixNano,
					Size:          oldFile.Size,
					IsDir:         oldFile.IsDir,
				})
			}

			changeAmount++
		}

		if len(changes) >= 64 && stream != nil {
			err := stream.Send(&remote.ChangeChunk{Changes: changes})
			if err != nil {
				return 0, errors.Wrap(err, "send changes")
			}

			changes = make([]*remote.Change, 0, 64)
		}
	}

	// Send the remaining changes
	if len(changes) > 0 && stream != nil {
		err := stream.Send(&remote.ChangeChunk{Changes: changes})
		if err != nil {
			return 0, errors.Wrap(err, "send changes")
		}
	}

	return changeAmount, nil
}

func walkDir(path string, ignoreMatcher gitignore.IgnoreParser, state map[string]*remote.Change) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		// We ignore errors here
		return
	}

	for _, f := range files {
		absolutePath := filepath.Join(path, f.Name())
		if ignoreMatcher != nil && ignoreMatcher.MatchesPath(absolutePath) {
			continue
		}

		// Stat is necessary here, because readdir does not follow symlinks and
		// IsDir() returns false for symlinked folders
		stat, err := os.Stat(absolutePath)
		if err != nil {
			// Woops file is not here anymore -> ignore error
			return
		}

		// Check if directory
		if stat.IsDir() {
			state[absolutePath] = &remote.Change{
				Path:  absolutePath,
				IsDir: true,
			}

			walkDir(absolutePath, ignoreMatcher, state)
		} else {
			state[absolutePath] = &remote.Change{
				Path:          absolutePath,
				Size:          stat.Size(),
				MtimeUnix:     stat.ModTime().Unix(),
				MtimeUnixNano: stat.ModTime().UnixNano(),
				IsDir:         false,
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
