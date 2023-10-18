package sync

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/loft-sh/devspace/helper/server/ignoreparser"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/loft-sh/notify"
	"github.com/pkg/errors"
)

var syncRetries = 5
var initialUpstreamBatchSize = 5000

const waitForMoreChangesTimeout = time.Minute

// Options holds the sync options
type Options struct {
	Polling          bool
	NoRecursiveWatch bool

	Exec []latest.SyncExec

	ResolveCommand func(command string, args []string) (string, []string, error)

	ExcludePaths         []string
	DownloadExcludePaths []string
	UploadExcludePaths   []string

	RestartContainer bool
	StartContainer   bool

	UploadBatchCmd  string
	UploadBatchArgs []string

	UpstreamLimit   int64
	DownstreamLimit int64
	Verbose         bool

	UpstreamDisabled   bool
	DownstreamDisabled bool

	InitialSyncCompareBy latest.InitialSyncCompareBy
	InitialSync          latest.InitialSyncStrategy

	Starter DelayedContainerStarter

	Log log.Logger
}

// Sync holds the necessary information for the syncing process
type Sync struct {
	ctx       context.Context
	cancelCtx context.CancelFunc

	LocalPath string
	Options   Options

	tree      notify.Tree
	fileIndex *fileIndex

	ignoreMatcher         ignoreparser.IgnoreParser
	downloadIgnoreMatcher ignoreparser.IgnoreParser
	uploadIgnoreMatcher   ignoreparser.IgnoreParser

	log log.Logger

	upstream   *upstream
	downstream *downstream

	stopOnce sync.Once

	onError chan error
	onDone  chan struct{}

	// Used for testing
	readyChan chan bool
}

// NewSync creates a new sync for the given local path
func NewSync(ctx context.Context, localPath string, options Options) (*Sync, error) {
	cancelCtx, cancel := context.WithCancel(ctx)

	// we have to resolve the real local path, because the watcher gives us the real path always
	realLocalPath, err := filepath.EvalSymlinks(localPath)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "eval symlinks")
	}

	absoluteLocalPath, err := filepath.Abs(realLocalPath)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "absolute path")
	}

	absoluteRealLocalPath, err := filepath.EvalSymlinks(absoluteLocalPath)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "eval symlinks")
	}

	// We exclude the sync log to prevent an endless loop in upstream
	newExcludes := []string{}
	newExcludes = append(newExcludes, ".devspace/")
	newExcludes = append(newExcludes, options.ExcludePaths...)
	options.ExcludePaths = newExcludes

	// Create sync structure
	s := &Sync{
		ctx:       cancelCtx,
		cancelCtx: cancel,

		LocalPath: absoluteRealLocalPath,
		Options:   options,

		fileIndex: newFileIndex(),
		log:       options.Log,
	}

	err = s.initIgnoreParsers()
	if err != nil {
		return nil, errors.Wrap(err, "init ignore parsers")
	}

	return s, nil
}

// Error handles a sync error
func (s *Sync) Error(err error) {
	s.log.Errorf("Sync Error on %s: %v", s.LocalPath, err)
}

// InitUpstream inits the upstream
func (s *Sync) InitUpstream(reader io.ReadCloser, writer io.WriteCloser) error {
	upstream, err := newUpstream(reader, writer, s)
	if err != nil {
		return errors.Wrap(err, "new upstream")
	}

	s.upstream = upstream
	return nil
}

// InitDownstream inits the downstream
func (s *Sync) InitDownstream(reader io.ReadCloser, writer io.WriteCloser) error {
	downstream, err := newDownstream(reader, writer, s)
	if err != nil {
		return errors.Wrap(err, "new upstream")
	}

	s.downstream = downstream
	return nil
}

// Start starts a new sync instance
func (s *Sync) Start(onInitUploadDone chan struct{}, onInitDownloadDone chan struct{}, onDone chan struct{}, onError chan error) error {
	s.onError = onError
	s.onDone = onDone

	// start pinging the underlying connection
	s.downstream.startPing(onDone)
	s.upstream.startPing(onDone)

	s.mainLoop(onInitUploadDone, onInitDownloadDone)
	return nil
}

func (s *Sync) initIgnoreParsers() error {
	if s.Options.ExcludePaths != nil {
		ignoreMatcher, err := ignoreparser.CompilePaths(s.Options.ExcludePaths, s.log)
		if err != nil {
			return errors.Wrap(err, "compile exclude paths")
		}

		s.ignoreMatcher = ignoreMatcher
	}

	if s.Options.DownloadExcludePaths != nil {
		ignoreMatcher, err := ignoreparser.CompilePaths(s.Options.DownloadExcludePaths, s.log)
		if err != nil {
			return errors.Wrap(err, "compile download exclude paths")
		}

		s.downloadIgnoreMatcher = ignoreMatcher
	}

	if s.Options.UploadExcludePaths != nil {
		ignoreMatcher, err := ignoreparser.CompilePaths(s.Options.UploadExcludePaths, s.log)
		if err != nil {
			return errors.Wrap(err, "compile upload exclude paths")
		}

		s.uploadIgnoreMatcher = ignoreMatcher
	}

	return nil
}

func (s *Sync) mainLoop(onInitUploadDone chan struct{}, onInitDownloadDone chan struct{}) {
	s.log.Info("Start syncing")

	// Start upstream as early as possible
	if !s.Options.UpstreamDisabled {
		go s.startUpstream()
	}

	// Start downstream and do initial sync
	go func() {
		err := s.initialSync(onInitUploadDone, onInitDownloadDone)
		if err != nil {
			s.Stop(errors.Wrap(err, "initial sync"))
			return
		}

		if !s.Options.DownstreamDisabled {
			s.startDownstream()
			s.Stop(nil)
		}
	}()
}

func (s *Sync) startUpstream() {
	defer s.Stop(nil)
	s.tree = notify.NewTree()

	// Set up a watchpoint listening for events within a directory tree rooted at specified directory
	watchPath := s.LocalPath + "/..."
	if s.Options.NoRecursiveWatch {
		watchPath = s.LocalPath
	}
	err := s.tree.Watch(watchPath, s.upstream.events, func(path string) bool {
		if s.ignoreMatcher == nil || s.ignoreMatcher.RequireFullScan() {
			return false
		}

		stat, err := os.Stat(path)
		if err != nil {
			return false
		}

		return s.ignoreMatcher.Matches(path[len(s.LocalPath):], stat.IsDir())
	}, notify.All)
	if err != nil {
		s.Stop(err)
		return
	}
	defer s.tree.Stop(s.upstream.events)
	if s.readyChan != nil {
		s.readyChan <- true
	}

	err = s.upstream.mainLoop()
	if err != nil {
		s.Stop(errors.Wrap(err, "upstream"))
	}
}

func (s *Sync) startDownstream() {
	defer s.Stop(nil)

	err := s.downstream.mainLoop()
	if err != nil {
		s.Stop(errors.Wrap(err, "downstream"))
	}
}

func (s *Sync) initialSync(onInitUploadDone chan struct{}, onInitDownloadDone chan struct{}) error {
	initialSync := newInitialSyncer(&initialSyncOptions{
		LocalPath: s.LocalPath,
		Strategy:  s.Options.InitialSync,
		CompareBy: s.Options.InitialSyncCompareBy,

		IgnoreMatcher:         s.ignoreMatcher,
		DownloadIgnoreMatcher: s.downloadIgnoreMatcher,
		UploadIgnoreMatcher:   s.uploadIgnoreMatcher,

		UpstreamDisabled:   s.Options.UpstreamDisabled,
		DownstreamDisabled: s.Options.DownstreamDisabled,
		FileIndex:          s.fileIndex,

		ApplyRemote: s.sendChangesToUpstream,
		ApplyLocal:  s.downstream.applyChanges,
		AddSymlink:  s.upstream.AddSymlink,
		Log:         s.log,

		UpstreamDone: func() {
			if !s.Options.UpstreamDisabled {
				for s.upstream.IsBusy() {
					time.Sleep(time.Millisecond * 100)
				}
			}

			// signal upstream that initial sync is done
			s.upstream.initialSyncCompletedMutex.Lock()
			s.upstream.initialSyncCompleted = true
			s.upstream.initialSyncCompletedMutex.Unlock()

			// wait until initial sync commands were executed
			if !s.Options.UpstreamDisabled {
				for s.upstream.IsInitialSyncing() {
					time.Sleep(time.Millisecond * 100)
				}
			}

			if onInitUploadDone != nil {
				s.log.Info("Upstream - Initial sync completed")
				close(onInitUploadDone)
			}
		},
		DownstreamDone: func() {
			if onInitDownloadDone != nil {
				s.log.Info("Downstream - Initial sync completed")
				close(onInitDownloadDone)
			}
		},
	})

	s.log.Debugf("Initial Sync - Retrieve Initial State")
	errChan := make(chan error)
	go func() {
		errChan <- s.downstream.populateFileMap()
	}()

	localState := make(map[string]*FileInformation)
	err := initialSync.CalculateLocalState(s.LocalPath, localState, false)
	if err != nil {
		<-errChan
		return err
	}

	err = <-errChan
	s.log.Debugf("Initial Sync - Done Retrieving Initial State")
	if err != nil {
		return errors.Wrap(err, "populate file map")
	}

	downloadChanges := make(map[string]*FileInformation)
	s.fileIndex.fileMapMutex.Lock()
	for key, element := range s.fileIndex.fileMap {
		if s.downloadIgnoreMatcher != nil && s.downloadIgnoreMatcher.Matches(element.Name, element.IsDirectory) {
			continue
		}
		if element.IsSymbolicLink {
			continue
		}

		downloadChanges[key] = element
	}
	s.fileIndex.fileMapMutex.Unlock()

	return initialSync.Run(downloadChanges, localState)
}

func (s *Sync) sendChangesToUpstream(changes []*FileInformation, remove bool) {
	for j := 0; j < len(changes); j += initialUpstreamBatchSize {
		// Wait till upstream channel is empty
		for len(s.upstream.events) > 0 {
			time.Sleep(time.Millisecond * 500)
		}

		// Now we send them to upstream
		sendBatch := make([]*FileInformation, 0, initialUpstreamBatchSize)
		s.fileIndex.fileMapMutex.Lock()

		for i := j; i < (j+initialUpstreamBatchSize) && i < len(changes); i++ {
			if remove {
				sendBatch = append(sendBatch, changes[i])
			} else if s.fileIndex.fileMap[changes[i].Name] == nil || !equalFilePermissions(changes[i].Mode, s.fileIndex.fileMap[changes[i].Name].Mode) || changes[i].Mtime != s.fileIndex.fileMap[changes[i].Name].Mtime || changes[i].Size != s.fileIndex.fileMap[changes[i].Name].Size {
				sendBatch = append(sendBatch, changes[i])
			}
		}

		s.fileIndex.fileMapMutex.Unlock()

		// We do this out of the fileIndex lock, because otherwise this could cause a deadlock
		// (Upstream waits in getfileInformationFromEvent and upstream.events buffer is full)
		for i := 0; i < len(sendBatch); i++ {
			s.upstream.isBusyMutex.Lock()
			s.upstream.isBusy = true
			s.upstream.events <- sendBatch[i]
			s.upstream.isBusyMutex.Unlock()
		}
	}
}

func equalFilePermissions(mode os.FileMode, mode2 os.FileMode) bool {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return true
	}

	return mode == mode2
}

// Stop stops the sync process
func (s *Sync) Stop(fatalError error) {
	s.stopOnce.Do(func() {
		s.cancelCtx()
		if s.upstream != nil {
			for _, symlink := range s.upstream.symlinks {
				symlink.Stop()
			}
			if s.upstream.writer != nil {
				s.upstream.writer.Close()
			}
			if s.upstream.reader != nil {
				s.upstream.reader.Close()
			}
			if s.upstream.conn != nil {
				s.upstream.conn.Close()
			}
		}

		if s.downstream != nil {
			if s.downstream.writer != nil {
				s.downstream.writer.Close()
			}
			if s.downstream.reader != nil {
				s.downstream.reader.Close()
			}
			if s.downstream.conn != nil {
				s.downstream.conn.Close()
			}
		}

		if fatalError != nil {
			s.Error(fatalError)

			// This needs to be rethought because we do not always kill the application here, would be better to have an error channel
			// or runtime error here
			if s.onError != nil {
				s.onError <- fatalError
			}
		}

		s.log.Debugf("Sync stopped")
		if s.onDone != nil {
			close(s.onDone)
		}
	})
}
