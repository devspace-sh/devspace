package sync

import (
	"github.com/loft-sh/devspace/helper/server/ignoreparser"
	"io"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/pkg/errors"
	"github.com/syncthing/notify"
)

var syncRetries = 5
var initialUpstreamBatchSize = 1000

// Options holds the sync options
type Options struct {
	ExcludePaths         []string
	DownloadExcludePaths []string
	UploadExcludePaths   []string

	RestartContainer bool

	FileChangeCmd  string
	FileChangeArgs []string

	DirCreateCmd  string
	DirCreateArgs []string

	UpstreamLimit   int64
	DownstreamLimit int64
	Verbose         bool

	UpstreamDisabled   bool
	DownstreamDisabled bool

	InitialSyncCompareBy latest.InitialSyncCompareBy
	InitialSync          latest.InitialSyncStrategy

	// These channels can be used to listen for certain sync events
	DownstreamInitialSyncDone chan bool
	UpstreamInitialSyncDone   chan bool
	SyncDone                  chan bool
	SyncError                 chan error

	Log log.Logger
}

// Sync holds the necessary information for the syncing process
type Sync struct {
	LocalPath string
	Options   Options

	fileIndex *fileIndex

	ignoreMatcher         ignoreparser.IgnoreParser
	downloadIgnoreMatcher ignoreparser.IgnoreParser
	uploadIgnoreMatcher   ignoreparser.IgnoreParser

	log log.Logger

	upstream   *upstream
	downstream *downstream

	silent   bool
	stopOnce sync.Once

	// Used for testing
	readyChan chan bool
}

// NewSync creates a new sync for the given
func NewSync(localPath string, options Options) (*Sync, error) {
	// we have to resolve the real local path, because the watcher gives us the real path always
	realLocalPath, err := filepath.EvalSymlinks(localPath)
	if err != nil {
		return nil, errors.Wrap(err, "eval symlinks")
	}

	absoluteLocalPath, err := filepath.Abs(realLocalPath)
	if err != nil {
		return nil, errors.Wrap(err, "absolute path")
	}

	if options.ExcludePaths == nil {
		options.ExcludePaths = make([]string, 0, 2)
	}

	// We exclude the sync log to prevent an endless loop in upstream
	options.ExcludePaths = append(options.ExcludePaths, ".devspace/")

	// Initialize log
	if options.Log == nil {
		options.Log = log.GetFileLogger("sync")
	}

	// Create sync structure
	s := &Sync{
		LocalPath: absoluteLocalPath,
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
func (s *Sync) Start() error {
	s.mainLoop()

	return nil
}

func (s *Sync) initIgnoreParsers() error {
	if s.Options.ExcludePaths != nil {
		ignoreMatcher, err := ignoreparser.CompilePaths(s.Options.ExcludePaths)
		if err != nil {
			return errors.Wrap(err, "compile exclude paths")
		}

		s.ignoreMatcher = ignoreMatcher
	}

	if s.Options.DownloadExcludePaths != nil {
		ignoreMatcher, err := ignoreparser.CompilePaths(s.Options.DownloadExcludePaths)
		if err != nil {
			return errors.Wrap(err, "compile download exclude paths")
		}

		s.downloadIgnoreMatcher = ignoreMatcher
	}

	if s.Options.UploadExcludePaths != nil {
		ignoreMatcher, err := ignoreparser.CompilePaths(s.Options.UploadExcludePaths)
		if err != nil {
			return errors.Wrap(err, "compile upload exclude paths")
		}

		s.uploadIgnoreMatcher = ignoreMatcher
	}

	return nil
}

func (s *Sync) mainLoop() {
	s.log.Info("Start syncing")

	// Start upstream as early as possible
	if s.Options.UpstreamDisabled == false {
		go s.startUpstream()
	}

	// Start downstream and do initial sync
	go func() {
		err := s.initialSync()
		if err != nil {
			s.Stop(errors.Wrap(err, "initial sync"))
			return
		}

		if s.Options.DownstreamDisabled == false {
			s.startDownstream()
			s.Stop(nil)
		}
	}()
}

func (s *Sync) startUpstream() {
	defer s.Stop(nil)

	// Set up a watchpoint listening for events within a directory tree rooted at specified directory
	err := notify.Watch(s.LocalPath+"/...", s.upstream.events, notify.All)
	if err != nil {
		s.Stop(err)
		return
	}

	defer notify.Stop(s.upstream.events)

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

func (s *Sync) initialSync() error {
	err := s.downstream.populateFileMap()
	if err != nil {
		return errors.Wrap(err, "populate file map")
	}

	downloadChanges := make(map[string]*FileInformation)
	s.fileIndex.fileMapMutex.Lock()
	for key, element := range s.fileIndex.fileMap {
		if element.IsSymbolicLink {
			continue
		}

		downloadChanges[key] = element
	}
	s.fileIndex.fileMapMutex.Unlock()

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
			if s.Options.UpstreamInitialSyncDone != nil {
				if s.Options.UpstreamDisabled == false {
					for len(s.upstream.events) > 0 || s.upstream.IsBusy() {
						time.Sleep(time.Millisecond * 100)
					}
				}

				s.log.Info("Upstream - Initial sync completed")
				close(s.Options.UpstreamInitialSyncDone)
			}
		},
		DownstreamDone: func() {
			if s.Options.DownstreamInitialSyncDone != nil {
				s.log.Info("Downstream - Initial sync completed")
				close(s.Options.DownstreamInitialSyncDone)
			}
		},
	})

	return initialSync.Run(downloadChanges)
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
			} else if s.fileIndex.fileMap[changes[i].Name] == nil || (s.fileIndex.fileMap[changes[i].Name] != nil && changes[i].Mtime > s.fileIndex.fileMap[changes[i].Name].Mtime) {
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

// Stop stops the sync process
func (s *Sync) Stop(fatalError error) {
	s.stopOnce.Do(func() {
		if s.upstream != nil && s.upstream.interrupt != nil {
			for _, symlink := range s.upstream.symlinks {
				symlink.Stop()
			}

			close(s.upstream.interrupt)
			if s.upstream.writer != nil {
				s.upstream.writer.Close()
			}
			if s.upstream.reader != nil {
				// Closing the reader is sometimes hanging on windows so we skip that
				if runtime.GOOS != "windows" {
					s.upstream.reader.Close()
				} else {
					go s.upstream.reader.Close()
				}
			}
		}

		if s.downstream != nil && s.downstream.interrupt != nil {
			close(s.downstream.interrupt)
			if s.downstream.writer != nil {
				s.downstream.writer.Close()
			}
			if s.downstream.reader != nil {
				// Closing the reader is sometimes hanging on windows so we skip that
				if runtime.GOOS != "windows" {
					s.downstream.reader.Close()
				} else {
					go s.downstream.reader.Close()
				}
			}
		}

		if fatalError != nil {
			s.Error(fatalError)

			// This needs to be rethought because we do not always kill the application here, would be better to have an error channel
			// or runtime error here
			if s.Options.SyncError != nil {
				s.Options.SyncError <- fatalError
			}
		}

		s.log.Infof("Sync stopped")
		if s.Options.SyncDone != nil {
			close(s.Options.SyncDone)
		}
	})
}
