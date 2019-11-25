package sync

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/analytics/cloudanalytics"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/sync/remote"
	"github.com/devspace-cloud/devspace/sync/util"

	"github.com/pkg/errors"
	"github.com/rjeczalik/notify"
	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/sirupsen/logrus"
)

var initialUpstreamBatchSize = 1000
var syncLog log.Logger

// Options holds the sync options
type Options struct {
	ExcludePaths         []string
	DownloadExcludePaths []string
	UploadExcludePaths   []string

	UpstreamLimit   int64
	DownstreamLimit int64
	Verbose         bool

	DownloadOnInitialSync bool

	// These channels can be used to listen for certain sync events
	DownstreamInitialSyncDone chan bool
	UpstreamInitialSyncDone   chan bool
	SyncDone                  chan bool

	Log log.Logger
}

// Sync holds the necessary information for the syncing process
type Sync struct {
	LocalPath string
	Options   *Options

	fileIndex *fileIndex

	ignoreMatcher         gitignore.IgnoreParser
	downloadIgnoreMatcher gitignore.IgnoreParser
	uploadIgnoreMatcher   gitignore.IgnoreParser

	log log.Logger

	upstream   *upstream
	downstream *downstream

	silent   bool
	stopOnce sync.Once

	// Used for testing
	errorChan chan error
	readyChan chan bool
}

// NewSync creates a new sync for the given
func NewSync(localPath string, options *Options) (*Sync, error) {
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

	// Initialize log, this is not thread safe !!!
	if options.Log == nil && syncLog == nil {
		// Check if syncLog already exists
		stat, err := os.Stat(log.Logdir + "sync.log")
		if err == nil || stat != nil {
			err = cleanupSyncLogs()
			if err != nil {
				return nil, errors.Wrap(err, "cleanup sync logs")
			}
		}

		syncLog = log.GetFileLogger("sync")
		syncLog.SetLevel(logrus.InfoLevel)
	}
	if options.Log == nil {
		options.Log = syncLog
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
	if s.errorChan != nil {
		s.errorChan <- err
	}
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
		ignoreMatcher, err := CompilePaths(s.Options.ExcludePaths)
		if err != nil {
			return errors.Wrap(err, "compile exclude paths")
		}

		s.ignoreMatcher = ignoreMatcher
	}

	if s.Options.DownloadExcludePaths != nil {
		ignoreMatcher, err := CompilePaths(s.Options.DownloadExcludePaths)
		if err != nil {
			return errors.Wrap(err, "compile download exclude paths")
		}

		s.downloadIgnoreMatcher = ignoreMatcher
	}

	if s.Options.UploadExcludePaths != nil {
		ignoreMatcher, err := CompilePaths(s.Options.UploadExcludePaths)
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
	go s.startUpstream()

	// Start downstream and do initial sync
	go func() {
		defer s.Stop(nil)
		err := s.initialSync()
		if err != nil {
			s.Stop(errors.Wrap(err, "initial sync"))
			return
		}

		s.log.Info("Initial sync completed")
		s.startDownstream()
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

	localChanges := make([]*FileInformation, 0, 10)
	fileMapClone := make(map[string]*FileInformation)

	s.fileIndex.fileMapMutex.Lock()
	for key, element := range s.fileIndex.fileMap {
		if element.IsSymbolicLink {
			continue
		}

		fileMapClone[key] = element
	}
	s.fileIndex.fileMapMutex.Unlock()

	err = s.diffServerClient(s.LocalPath, &localChanges, fileMapClone, false)
	if err != nil {
		return errors.Wrap(err, "diff server client")
	}

	// Upstream initial sync
	go func() {
		// Remove remote files that are not there locally
		if s.Options.DownloadOnInitialSync == false && len(fileMapClone) > 0 {
			remoteChanges := make([]*FileInformation, 0, len(fileMapClone))
			for _, element := range fileMapClone {
				remoteChanges = append(remoteChanges, &FileInformation{
					Name:        element.Name,
					IsDirectory: element.IsDirectory,
				})
			}

			s.sendChangesToUpstream(remoteChanges, true)
		}

		s.sendChangesToUpstream(localChanges, false)
		if s.Options.UpstreamInitialSyncDone != nil {
			close(s.Options.UpstreamInitialSyncDone)
		}
	}()

	// Download changes if enabled
	if s.Options.DownloadOnInitialSync && len(fileMapClone) > 0 {
		remoteChanges := make([]*remote.Change, 0, len(fileMapClone))
		for _, element := range fileMapClone {
			remoteChanges = append(remoteChanges, &remote.Change{
				ChangeType:    remote.ChangeType_CHANGE,
				Path:          element.Name,
				MtimeUnix:     element.Mtime,
				MtimeUnixNano: element.MtimeNano,
				Size:          element.Size,
				IsDir:         element.IsDirectory,
			})
		}

		err = s.downstream.applyChanges(remoteChanges)
		if err != nil {
			return errors.Wrap(err, "apply changes")
		}
	}

	if s.Options.DownstreamInitialSyncDone != nil {
		close(s.Options.DownstreamInitialSyncDone)
	}

	return nil
}

func (s *Sync) diffServerClient(absPath string, sendChanges *[]*FileInformation, downloadChanges map[string]*FileInformation, dontSend bool) error {
	relativePath := getRelativeFromFullPath(absPath, s.LocalPath)

	// We skip files that are suddenly not there anymore
	stat, err := os.Stat(absPath)
	if err != nil {
		return nil
	}

	delete(downloadChanges, relativePath)

	// Exclude changes on the upload exclude list
	if s.uploadIgnoreMatcher != nil {
		if util.MatchesPath(s.uploadIgnoreMatcher, relativePath, stat.IsDir()) {
			s.fileIndex.fileMapMutex.Lock()
			// Add to file map and prevent download if local file is newer than the remote one
			if s.fileIndex.fileMap[relativePath] != nil && s.fileIndex.fileMap[relativePath].Mtime < stat.ModTime().Unix() {
				// Add it to the fileMap
				s.fileIndex.fileMap[relativePath] = &FileInformation{
					Name:        relativePath,
					Mtime:       stat.ModTime().Unix(),
					MtimeNano:   stat.ModTime().UnixNano(),
					Size:        stat.Size(),
					IsDirectory: stat.IsDir(),
				}
			}
			s.fileIndex.fileMapMutex.Unlock()

			dontSend = true
		}
	}

	// Check for symlinks
	if dontSend == false {
		// Retrieve the real stat instead of the symlink one
		lstat, err := os.Lstat(absPath)
		if err == nil && lstat.Mode()&os.ModeSymlink != 0 {
			stat, err = s.upstream.AddSymlink(relativePath, absPath)
			if err != nil {
				return err
			}
			if stat == nil {
				return nil
			}

			s.log.Infof("Symlink found at %s", absPath)
		} else if err != nil {
			return nil
		}
	}

	if stat.IsDir() {
		return s.diffDir(absPath, stat, sendChanges, downloadChanges, dontSend)
	}

	if dontSend == false {
		s.fileIndex.fileMapMutex.Lock()
		shouldUpload := shouldUpload(relativePath, stat, s, true)
		s.fileIndex.fileMapMutex.Unlock()
		if shouldUpload {
			// Add file to upload
			*sendChanges = append(*sendChanges, &FileInformation{
				Name:        relativePath,
				Mtime:       stat.ModTime().Unix(),
				Size:        stat.Size(),
				IsDirectory: false,
			})
		}
	}

	return nil
}

func (s *Sync) diffDir(filepath string, stat os.FileInfo, sendChanges *[]*FileInformation, downloadChanges map[string]*FileInformation, dontSend bool) error {
	relativePath := getRelativeFromFullPath(filepath, s.LocalPath)
	files, err := ioutil.ReadDir(filepath)

	if err != nil {
		s.log.Infof("Couldn't read dir %s: %v", filepath, err)
		return nil
	}

	if len(files) == 0 && relativePath != "" && dontSend == false {
		s.fileIndex.fileMapMutex.Lock()
		shouldUpload := shouldUpload(relativePath, stat, s, true)
		s.fileIndex.fileMapMutex.Unlock()

		if shouldUpload {
			*sendChanges = append(*sendChanges, &FileInformation{
				Name:        relativePath,
				Mtime:       stat.ModTime().Unix(),
				Size:        stat.Size(),
				IsDirectory: true,
			})
		}
	}

	for _, f := range files {
		if err := s.diffServerClient(path.Join(filepath, f.Name()), sendChanges, downloadChanges, dontSend); err != nil {
			return errors.Wrap(err, f.Name())
		}
	}

	return nil
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
			s.upstream.events <- sendBatch[i]
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
				// Closing the reader is hanging on windows so we skip that
				// s.upstream.reader.Close()
			}
		}

		if s.downstream != nil && s.downstream.interrupt != nil {
			close(s.downstream.interrupt)
			if s.downstream.writer != nil {
				s.downstream.writer.Close()
			}
			if s.downstream.reader != nil {
				// Closing the reader is hanging on windows so we skip that
				// s.downstream.reader.Close()
			}
		}

		s.log.Infof("Sync stopped")
		if s.Options.SyncDone != nil {
			close(s.Options.SyncDone)
		}

		if fatalError != nil {
			s.Error(fatalError)

			// This needs to be rethought because we do not always kill the application here, would be better to have an error channel
			// or runtime error here
			sendError := fmt.Errorf("Fatal sync error: %v. For more information check .devspace/logs/sync.log", fatalError)
			log.GetInstance().Error(sendError)
			cloudanalytics.SendCommandEvent(sendError)
			os.Exit(1)
		}
	})
}
