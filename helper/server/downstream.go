package server

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/devspace/helper/util/pingtimeout"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"

	"github.com/loft-sh/devspace/helper/server/ignoreparser"
	"github.com/loft-sh/devspace/helper/util/stderrlog"
	"github.com/loft-sh/notify"

	"github.com/loft-sh/devspace/helper/remote"
	"github.com/loft-sh/devspace/helper/util"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const rescanPeriod = time.Minute * 15

// DownstreamOptions holds the options for the downstream server
type DownstreamOptions struct {
	RemotePath       string
	ExcludePaths     []string
	ExitOnClose      bool
	NoRecursiveWatch bool
	Throttle         int64

	Polling bool
	Ping    bool
}

// StartDownstreamServer starts a new downstream server with the given reader and writer
func StartDownstreamServer(reader io.Reader, writer io.Writer, options *DownstreamOptions) error {
	pipe := util.NewStdStreamJoint(reader, writer, options.ExitOnClose)
	lis := util.NewStdinListener()
	done := make(chan error)

	// Compile ignore paths
	ignoreMatcher, err := ignoreparser.CompilePaths(options.ExcludePaths, logpkg.Discard)
	if err != nil {
		return errors.Wrap(err, "compile paths")
	}

	go func() {
		s := grpc.NewServer()
		downStream := &Downstream{
			options:       options,
			ignoreMatcher: ignoreMatcher,
			events:        make(chan notify.EventInfo, 1000),
			changes:       map[string]bool{},
			ping:          &pingtimeout.PingTimeout{},
		}

		if options.Ping {
			doneChan := make(chan struct{})
			defer close(doneChan)
			downStream.ping.Start(doneChan)
		}

		remote.RegisterDownstreamServer(s, downStream)
		reflection.Register(s)

		// start watcher if this we should use it
		watchStop := make(chan struct{})
		if !options.Polling {
			stderrlog.Infof("Use inotify as watching method in container")

			go func() {
				// set up a watchpoint listening for events within a directory tree rooted at specified directory
				watchPath := options.RemotePath + "/..."
				if options.NoRecursiveWatch {
					watchPath = options.RemotePath
				}

				err := notify.WatchWithFilter(watchPath, downStream.events, func(s string) bool {
					if ignoreMatcher == nil || ignoreMatcher.RequireFullScan() {
						return false
					}

					stat, err := os.Stat(s)
					if err != nil {
						return false
					}

					return ignoreMatcher.Matches(s[len(options.RemotePath):], stat.IsDir())
				}, notify.All)
				if err != nil {
					log.Fatalf("error watching path %s: %v", options.RemotePath, err)
					return
				}
				defer notify.Stop(downStream.events)

				// start the watch loop
				downStream.watch(watchStop)
			}()
		} else {
			stderrlog.Infof("Use polling as watching method in container")
		}

		done <- s.Serve(lis)
		close(watchStop)
	}()

	lis.Ready(pipe)
	return <-done
}

// Downstream is the implementation for the downstream server
type Downstream struct {
	remote.UnimplementedDownstreamServer

	options *DownstreamOptions

	// ignore matcher is the ignore matcher which matches against excluded files and paths
	ignoreMatcher ignoreparser.IgnoreParser

	// watchedFiles is a memory map of the previous state of the changes function
	watchedFiles map[string]*remote.Change

	// events is the event stream if we watch for changes
	events chan notify.EventInfo

	// changesMutex is used to protect changes
	changesMutex sync.Mutex

	// changes is a map of changed paths
	changes map[string]bool

	// lastRescan is used to rescan the complete path from time to time
	lastRescan *time.Time

	// ping is used to determine if we still have an alive connection
	ping *pingtimeout.PingTimeout
}

// Download sends the file at the temp download location to the client
func (d *Downstream) Download(stream remote.Downstream_DownloadServer) error {
	filesToCompress := make([]string, 0, 128)
	for {
		paths, err := stream.Recv()
		if paths != nil {
			filesToCompress = append(filesToCompress, paths.Paths...)
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

	reader.Close()
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
		if _, ok := writtenFiles[path]; !ok {
			err := recursiveTar(d.options.RemotePath, path, writtenFiles, tarWriter, true)
			if err != nil {
				return errors.Wrapf(err, "compress %s", path)
			}
		}
	}

	return nil
}

// Ping returns empty
func (d *Downstream) Ping(context.Context, *remote.Empty) (*remote.Empty, error) {
	if d.ping != nil {
		d.ping.Ping()
	}

	return &remote.Empty{}, nil
}

// ChangesCount returns the amount of changes on the remote side
func (d *Downstream) ChangesCount(context.Context, *remote.Empty) (*remote.ChangeAmount, error) {
	newState := make(map[string]*remote.Change)
	throttle := time.Duration(d.options.Throttle) * time.Millisecond

	// Walk through the dir
	if d.options.Polling {
		walkDir(d.options.RemotePath, d.options.RemotePath, d.ignoreMatcher, newState, d.options.NoRecursiveWatch, throttle)
	}

	changeAmount := int64(0)
	if d.options.Polling {
		var err error
		changeAmount, err = streamChanges(d.options.RemotePath, d.watchedFiles, newState, nil, throttle)
		if err != nil {
			return nil, errors.Wrap(err, "count changes")
		}
	} else {
		d.changesMutex.Lock()
		// if rescan is not set we make sure that we say that there
		// are changes
		if d.lastRescan == nil {
			changeAmount = int64(1)
		} else {
			changeAmount = int64(len(d.changes))
		}
		d.changesMutex.Unlock()
	}

	return &remote.ChangeAmount{
		Amount: changeAmount,
	}, nil
}

func (d *Downstream) getWatchState() map[string]*remote.Change {
	d.changesMutex.Lock()
	defer d.changesMutex.Unlock()

	var (
		now          = time.Now()
		shouldRescan = d.watchedFiles == nil || d.lastRescan == nil || d.lastRescan.Add(rescanPeriod).Before(now)
		changeAmount = len(d.changes)
	)

	if changeAmount > 100 || shouldRescan {
		// we rescan so reset all changes
		d.changes = map[string]bool{}
		d.lastRescan = &now

		newState := make(map[string]*remote.Change)
		walkDir(d.options.RemotePath, d.options.RemotePath, d.ignoreMatcher, newState, d.options.NoRecursiveWatch, 0)
		return newState
	} else if changeAmount == 0 {
		return nil
	}

	// copy state from old
	newState := copyState(d.watchedFiles)

	// copy changes
	changes := []string{}
	for k := range d.changes {
		changes = append(changes, k)
	}
	d.changes = map[string]bool{}

	// apply changes
	for _, change := range changes {
		d.applyChange(newState, change)
	}

	return newState
}

// Changes retrieves all changes from the watch path
func (d *Downstream) Changes(empty *remote.Empty, stream remote.Downstream_ChangesServer) error {
	newState := make(map[string]*remote.Change)
	throttle := time.Duration(d.options.Throttle) * time.Millisecond

	// Walk through the dir
	if !d.options.Polling {
		newState = d.getWatchState()
	} else {
		walkDir(d.options.RemotePath, d.options.RemotePath, d.ignoreMatcher, newState, d.options.NoRecursiveWatch, throttle)
	}

	if newState != nil {
		_, err := streamChanges(d.options.RemotePath, d.watchedFiles, newState, stream, throttle)
		if err != nil {
			return errors.Wrap(err, "stream changes")
		}

		d.watchedFiles = newState
	}
	return nil
}

func (d *Downstream) watch(stopChan chan struct{}) {
	for {
		select {
		case <-stopChan:
			return
		case event, ok := <-d.events:
			if !ok {
				return
			}

			d.changesMutex.Lock()
			// re-sync if overflow
			if len(d.events) >= 999 {
				d.lastRescan = nil
			} else {
				// check if parent folder might be already in changes then skip
				// this saves us a lot of folder crawling later on
				parts := strings.Split(filepath.ToSlash(event.Path()), "/")
				found := false
				for i := len(parts) - 1; i > 0; i-- {
					path := strings.Join(parts[:i], "/")
					if d.changes[path] {
						found = true
						break
					}
				}
				if !found {
					d.changes[event.Path()] = true
				}
			}
			d.changesMutex.Unlock()
		}
	}
}

func (d *Downstream) applyChange(newState map[string]*remote.Change, fullPath string) {
	fullPath = strings.TrimSuffix(fullPath, "/")

	relativePath := fullPath[len(d.options.RemotePath):]

	// in any case we mark this part of the tree as dirty and delete it
	if old, ok := d.watchedFiles[fullPath]; ok {
		if old.IsDir {
			deletePathFromState(newState, fullPath)
		} else {
			delete(newState, fullPath)
		}
	}

	// check if the path still exists
	stat, err := os.Stat(fullPath)
	if err != nil {
		return
	} else if d.ignoreMatcher != nil && !d.ignoreMatcher.RequireFullScan() && d.ignoreMatcher.Matches(relativePath, stat.IsDir()) {
		return
	}

	if stat.IsDir() {
		if d.ignoreMatcher == nil || !d.ignoreMatcher.RequireFullScan() || !d.ignoreMatcher.Matches(relativePath, true) {
			newState[fullPath] = &remote.Change{
				Path:  fullPath,
				IsDir: true,
			}
		}

		walkDir(d.options.RemotePath, fullPath, d.ignoreMatcher, newState, d.options.NoRecursiveWatch, time.Duration(d.options.Throttle)*time.Millisecond)
	} else {
		if d.ignoreMatcher == nil || !d.ignoreMatcher.RequireFullScan() || !d.ignoreMatcher.Matches(relativePath, false) {
			newState[fullPath] = &remote.Change{
				Path:          fullPath,
				Size:          stat.Size(),
				MtimeUnix:     stat.ModTime().Unix(),
				MtimeUnixNano: stat.ModTime().UnixNano(),
				Mode:          uint32(stat.Mode()),
				IsDir:         false,
			}
		}
	}
}

func deletePathFromState(state map[string]*remote.Change, path string) {
	for k := range state {
		if strings.HasPrefix(k, path+"/") || k == path {
			delete(state, k)
		}
	}
}

func copyState(state map[string]*remote.Change) map[string]*remote.Change {
	newState := make(map[string]*remote.Change, len(state))
	for k, v := range state {
		newState[k] = &remote.Change{
			ChangeType:    v.ChangeType,
			Path:          v.Path,
			MtimeUnix:     v.MtimeUnix,
			MtimeUnixNano: v.MtimeUnixNano,
			Size:          v.Size,
			Mode:          v.Mode,
			IsDir:         v.IsDir,
		}
	}

	return newState
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
						Mode:          newFile.Mode,
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
					Mode:          newFile.Mode,
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

		if _, ok := newState[oldFile.Path]; !ok {
			if stream != nil {
				changes = append(changes, &remote.Change{
					ChangeType:    remote.ChangeType_DELETE,
					Path:          oldFile.Path[len(basePath):],
					MtimeUnix:     oldFile.MtimeUnix,
					MtimeUnixNano: oldFile.MtimeUnixNano,
					Size:          oldFile.Size,
					Mode:          oldFile.Mode,
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

func walkDir(basePath string, path string, ignoreMatcher ignoreparser.IgnoreParser, state map[string]*remote.Change, noRecursive bool, throttle time.Duration) {
	files, err := os.ReadDir(path)
	if err != nil {
		// We ignore errors here
		return
	}

	for _, dirEntry := range files {
		f, err := dirEntry.Info()
		if err != nil {
			continue
		}

		absolutePath := filepath.Join(path, f.Name())
		if fsutil.IsRecursiveSymlink(f, absolutePath) {
			continue
		}

		// Stat is necessary here, because readdir does not follow symlinks and
		// IsDir() returns false for symlinked folders
		stat, err := os.Stat(absolutePath)
		if err != nil {
			// Woops file is not here anymore -> ignore error
			continue
		}

		// Check if ignored
		if ignoreMatcher != nil && !ignoreMatcher.RequireFullScan() && ignoreMatcher.Matches(absolutePath[len(basePath):], stat.IsDir()) {
			continue
		}

		// should throttle?
		if throttle != 0 && len(state)%100 == 0 {
			time.Sleep(throttle)
		}

		// Check if directory
		if stat.IsDir() {
			// Check if not ignored
			if ignoreMatcher == nil || !ignoreMatcher.RequireFullScan() || !ignoreMatcher.Matches(absolutePath[len(basePath):], true) {
				state[absolutePath] = &remote.Change{
					Path:  absolutePath,
					IsDir: true,
				}
			}

			if !noRecursive {
				walkDir(basePath, absolutePath, ignoreMatcher, state, noRecursive, throttle)
			}
		} else {
			// Check if not ignored
			if ignoreMatcher == nil || !ignoreMatcher.RequireFullScan() || !ignoreMatcher.Matches(absolutePath[len(basePath):], false) {
				state[absolutePath] = &remote.Change{
					Path:          absolutePath,
					Size:          stat.Size(),
					MtimeUnix:     stat.ModTime().Unix(),
					MtimeUnixNano: stat.ModTime().UnixNano(),
					Mode:          uint32(stat.Mode()),
					IsDir:         false,
				}
			}
		}
	}
}
