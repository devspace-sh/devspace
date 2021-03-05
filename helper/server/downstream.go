package server

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"github.com/loft-sh/devspace/helper/server/ignoreparser"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/loft-sh/devspace/helper/remote"
	"github.com/loft-sh/devspace/helper/util"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// DownstreamOptions holds the options for the downstream server
type DownstreamOptions struct {
	RemotePath   string
	ExcludePaths []string
	ExitOnClose  bool
	Throttle     int64
}

// StartDownstreamServer starts a new downstream server with the given reader and writer
func StartDownstreamServer(reader io.Reader, writer io.Writer, options *DownstreamOptions) error {
	pipe := util.NewStdStreamJoint(reader, writer, options.ExitOnClose)
	lis := util.NewStdinListener()
	done := make(chan error)

	// Compile ignore paths
	ignoreMatcher, err := ignoreparser.CompilePaths(options.ExcludePaths)
	if err != nil {
		return errors.Wrap(err, "compile paths")
	}

	go func() {
		s := grpc.NewServer()

		remote.RegisterDownstreamServer(s, &Downstream{
			options:       options,
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
	options *DownstreamOptions

	// ignore matcher is the ignore matcher which matches against excluded files and paths
	ignoreMatcher ignoreparser.IgnoreParser

	// watchedFiles is a memory map of the previous state of the changes function
	watchedFiles map[string]*remote.Change
}

// Download sends the file at the temp download location to the client
func (d *Downstream) Download(stream remote.Downstream_DownloadServer) error {
	filesToCompress := make([]string, 0, 128)
	for {
		paths, err := stream.Recv()
		if paths != nil {
			for _, path := range paths.Paths {
				filesToCompress = append(filesToCompress, path)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	// Create os pipe
	reader, writer := io.Pipe()

	// Compress archive and send at the same time
	errorChan := make(chan error)
	go func() {
		errorChan <- d.compress(writer, filesToCompress)
	}()

	// Send compressed archive to client
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
			break
		} else if err != nil {
			return errors.Wrap(err, "read file")
		}
	}

	return <-errorChan
}

// Compress compresses the given files and folders into a tar archive
func (d *Downstream) compress(writer io.WriteCloser, files []string) error {
	defer writer.Close()

	// Use compression
	gw := gzip.NewWriter(writer)
	defer gw.Close()

	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	writtenFiles := make(map[string]bool)
	for _, path := range files {
		if _, ok := writtenFiles[path]; ok == false {
			err := recursiveTar(d.options.RemotePath, path, writtenFiles, tarWriter, true)
			if err != nil {
				return errors.Wrapf(err, "compress %s", path)
			}
		}
	}

	return nil
}

// ChangesCount returns the amount of changes on the remote side
func (d *Downstream) ChangesCount(context.Context, *remote.Empty) (*remote.ChangeAmount, error) {
	newState := make(map[string]*remote.Change)
	throttle := time.Duration(d.options.Throttle) * time.Millisecond

	// Walk through the dir
	walkDir(d.options.RemotePath, d.options.RemotePath, d.ignoreMatcher, newState, throttle)

	changeAmount, err := streamChanges(d.options.RemotePath, d.watchedFiles, newState, nil, throttle)
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
	throttle := time.Duration(d.options.Throttle) * time.Millisecond

	// Walk through the dir
	walkDir(d.options.RemotePath, d.options.RemotePath, d.ignoreMatcher, newState, throttle)

	_, err := streamChanges(d.options.RemotePath, d.watchedFiles, newState, stream, throttle)
	if err != nil {
		return errors.Wrap(err, "stream changes")
	}

	d.watchedFiles = newState
	return nil
}

func streamChanges(basePath string, oldState map[string]*remote.Change, newState map[string]*remote.Change, stream remote.Downstream_ChangesServer, throttle time.Duration) (int64, error) {
	changeAmount := int64(0)
	if oldState == nil {
		oldState = make(map[string]*remote.Change)
	}

	// Compare new -> old
	changes := make([]*remote.Change, 0, 64)
	counter := int64(0)
	for _, newFile := range newState {
		counter++
		if throttle != 0 && counter%2000 == 0 {
			time.Sleep(throttle)
		}

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
		counter++
		if throttle != 0 && counter%2000 == 0 {
			time.Sleep(throttle)
		}

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

func walkDir(basePath string, path string, ignoreMatcher ignoreparser.IgnoreParser, state map[string]*remote.Change, throttle time.Duration) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		// We ignore errors here
		return
	}

	for _, f := range files {
		absolutePath := filepath.Join(path, f.Name())

		// Stat is necessary here, because readdir does not follow symlinks and
		// IsDir() returns false for symlinked folders
		stat, err := os.Stat(absolutePath)
		if err != nil {
			// Woops file is not here anymore -> ignore error
			continue
		}

		// Check if ignored
		if ignoreMatcher != nil && ignoreMatcher.HasNegatePatterns() == false && util.MatchesPath(ignoreMatcher, absolutePath[len(basePath):], stat.IsDir()) {
			continue
		}

		// should throttle?
		if throttle != 0 && len(state)%100 == 0 {
			time.Sleep(throttle)
		}

		// Check if directory
		if stat.IsDir() {
			// Check if not ignored
			if ignoreMatcher == nil || ignoreMatcher.HasNegatePatterns() == false || util.MatchesPath(ignoreMatcher, absolutePath[len(basePath):], true) == false {
				state[absolutePath] = &remote.Change{
					Path:  absolutePath,
					IsDir: true,
				}
			}

			walkDir(basePath, absolutePath, ignoreMatcher, state, throttle)
		} else {
			// Check if not ignored
			if ignoreMatcher == nil || ignoreMatcher.HasNegatePatterns() == false || util.MatchesPath(ignoreMatcher, absolutePath[len(basePath):], false) == false {
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
}
