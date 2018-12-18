package sync

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/juju/errors"
	"github.com/rjeczalik/notify"
	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/sirupsen/logrus"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

var initialUpstreamBatchSize = 1000
var syncLog log.Logger

//StartAck signals to the user that the sync process is starting
const StartAck string = "START"

//EndAck signals to the user that the sync process is done
const EndAck string = "DONE"

//ErrorAck signals to the user that an error occurred
const ErrorAck string = "ERROR"

// SyncConfig holds the necessary information for the syncing process
type SyncConfig struct {
	Kubectl              *kubernetes.Clientset
	Pod                  *k8sv1.Pod
	Container            *k8sv1.Container
	WatchPath            string
	DestPath             string
	ExcludePaths         []string
	DownloadExcludePaths []string
	UploadExcludePaths   []string
	UpstreamLimit        int64
	DownstreamLimit      int64
	Verbose              bool

	fileIndex *fileIndex

	ignoreMatcher         gitignore.IgnoreParser
	downloadIgnoreMatcher gitignore.IgnoreParser
	uploadIgnoreMatcher   gitignore.IgnoreParser

	log *logrus.Logger

	upstream   *upstream
	downstream *downstream

	silent   bool
	stopOnce sync.Once

	// Used for testing
	testing   bool
	errorChan chan error
	readyChan chan bool
}

// Logf prints the given information to the synclog with context data
func (s *SyncConfig) Logf(format string, args ...interface{}) {
	if s.silent == false {
		if s.Pod != nil {
			syncLog.WithKey("pod", s.Pod.Name).WithKey("local", s.WatchPath).WithKey("container", s.DestPath).Infof(format, args...)
		} else {
			syncLog.WithKey("local", s.WatchPath).WithKey("container", s.DestPath).Infof(format, args...)
		}
	}
}

// Logln prints the given information to the synclog with context data
func (s *SyncConfig) Logln(line interface{}) {
	if s.silent == false {
		if s.Pod != nil {
			syncLog.WithKey("pod", s.Pod.Name).WithKey("local", s.WatchPath).WithKey("container", s.DestPath).Info(line)
		} else {
			syncLog.
				WithKey("local",
					s.WatchPath).
				WithKey("container", s.DestPath).
				Info(line)
		}
	}
}

// Error handles a sync error with context
func (s *SyncConfig) Error(err error) {
	if s.Pod != nil {
		syncLog.WithKey("pod", s.Pod.Name).WithKey("local", s.WatchPath).WithKey("container", s.DestPath).Errorf("Error: %v, Stack: %v", err, errors.ErrorStack(err))
	} else {
		syncLog.WithKey("local", s.WatchPath).WithKey("container", s.DestPath).Errorf("Error: %v, Stack: %v", err, errors.ErrorStack(err))
	}

	if s.errorChan != nil {
		s.errorChan <- err
	}
}

func (s *SyncConfig) setup() error {
	// we have to resolve the real local path, because the watcher gives us the real path always
	realLocalPath, err := filepath.EvalSymlinks(s.WatchPath)
	if err != nil {
		return errors.Trace(err)
	}

	s.WatchPath = realLocalPath

	if s.ExcludePaths == nil {
		s.ExcludePaths = make([]string, 0, 2)
	}

	// We exclude the sync log to prevent an endless loop in upstream
	s.fileIndex = newFileIndex()
	s.ExcludePaths = append(s.ExcludePaths, "/.devspace/logs")

	if syncLog == nil {
		// Check if syncLog already exists
		stat, err := os.Stat(log.Logdir + "sync.log")

		if err == nil || stat != nil {
			err = cleanupSyncLogs()

			if err != nil {
				return errors.Trace(err)
			}
		}

		syncLog = log.GetFileLogger("sync")
		syncLog.SetLevel(logrus.InfoLevel)
	}

	err = s.initIgnoreParsers()
	if err != nil {
		return errors.Trace(err)
	}

	// Init upstream
	s.upstream = &upstream{
		config: s,
	}

	// Init downstream
	s.downstream = &downstream{
		config: s,
	}

	return nil
}

// Start starts a new sync instance
func (s *SyncConfig) Start() error {
	err := s.setup()
	if err != nil {
		return errors.Trace(err)
	}

	err = s.upstream.start()
	if err != nil {
		return errors.Trace(err)
	}

	err = s.downstream.start()
	if err != nil {
		s.Stop(nil)
		return errors.Trace(err)
	}

	go s.mainLoop()

	return nil
}

func (s *SyncConfig) initIgnoreParsers() error {
	if s.ExcludePaths != nil {
		ignoreMatcher, err := compilePaths(s.ExcludePaths)
		if err != nil {
			return errors.Trace(err)
		}

		s.ignoreMatcher = ignoreMatcher
	}

	if s.DownloadExcludePaths != nil {
		ignoreMatcher, err := compilePaths(s.DownloadExcludePaths)
		if err != nil {
			return errors.Trace(err)
		}

		s.downloadIgnoreMatcher = ignoreMatcher
	}

	if s.UploadExcludePaths != nil {
		ignoreMatcher, err := compilePaths(s.UploadExcludePaths)
		if err != nil {
			return errors.Trace(err)
		}

		s.uploadIgnoreMatcher = ignoreMatcher
	}

	return nil
}

func (s *SyncConfig) mainLoop() {
	s.Logf("[Sync] Start syncing")

	// Start upstream as early as possible
	go s.startUpstream()

	// Start downstream and do initial sync
	go func() {
		defer s.Stop(nil)

		err := s.initialSync()
		if err != nil {
			s.Stop(err)
			return
		}

		s.Logf("[Sync] Initial sync completed")
		s.startDownstream()
	}()
}

func (s *SyncConfig) startUpstream() {
	defer s.Stop(nil)

	// Set up a watchpoint listening for events within a directory tree rooted at specified directory
	err := notify.Watch(s.WatchPath+"/...", s.upstream.events, notify.All)
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
		s.Stop(err)
	}
}

func (s *SyncConfig) startDownstream() {
	defer s.Stop(nil)

	err := s.downstream.mainLoop()
	if err != nil {
		s.Stop(err)
	}
}

func (s *SyncConfig) initialSync() error {
	err := s.downstream.populateFileMap()
	if err != nil {
		return errors.Trace(err)
	}

	localChanges := make([]*fileInformation, 0, 10)
	fileMapClone := make(map[string]*fileInformation)

	s.fileIndex.fileMapMutex.Lock()
	for key, element := range s.fileIndex.fileMap {
		if element.IsSymbolicLink {
			continue
		}

		fileMapClone[key] = element
	}
	s.fileIndex.fileMapMutex.Unlock()

	err = s.diffServerClient(s.WatchPath, &localChanges, fileMapClone, false)
	if err != nil {
		return errors.Trace(err)
	}

	if len(localChanges) > 0 {
		go s.sendChangesToUpstream(localChanges)
	}

	if len(fileMapClone) > 0 {
		remoteChanges := make([]*fileInformation, 0, len(fileMapClone))
		for _, element := range fileMapClone {
			remoteChanges = append(remoteChanges, element)
		}

		err = s.downstream.applyChanges(remoteChanges, nil)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (s *SyncConfig) diffServerClient(absPath string, sendChanges *[]*fileInformation, downloadChanges map[string]*fileInformation, dontSend bool) error {
	relativePath := getRelativeFromFullPath(absPath, s.WatchPath)
	stat, err := os.Stat(absPath)

	// We skip files that are suddenly not there anymore
	if err != nil {
		return nil
	}

	delete(downloadChanges, relativePath)

	// Exclude changes on the upload exclude list
	if s.uploadIgnoreMatcher != nil {
		if s.uploadIgnoreMatcher.MatchesPath(relativePath) {
			s.fileIndex.fileMapMutex.Lock()
			// Add to file map and prevent download if local file is newer than the remote one
			if s.fileIndex.fileMap[relativePath] != nil && s.fileIndex.fileMap[relativePath].Mtime < roundMtime(stat.ModTime()) {
				// Add it to the fileMap
				s.fileIndex.fileMap[relativePath] = &fileInformation{
					Name:        relativePath,
					Mtime:       roundMtime(stat.ModTime()),
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

			s.Logf("Symlink at %s", absPath)
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
			*sendChanges = append(*sendChanges, &fileInformation{
				Name:        relativePath,
				Mtime:       roundMtime(stat.ModTime()),
				Size:        stat.Size(),
				IsDirectory: false,
			})
		}
	}

	return nil
}

func (s *SyncConfig) diffDir(filepath string, stat os.FileInfo, sendChanges *[]*fileInformation, downloadChanges map[string]*fileInformation, dontSend bool) error {
	relativePath := getRelativeFromFullPath(filepath, s.WatchPath)
	files, err := ioutil.ReadDir(filepath)

	if err != nil {
		s.Logf("[Upstream] Couldn't read dir %s: %v", filepath, err)
		return nil
	}

	if len(files) == 0 && relativePath != "" && dontSend == false {
		s.fileIndex.fileMapMutex.Lock()
		shouldUpload := shouldUpload(relativePath, stat, s, true)
		s.fileIndex.fileMapMutex.Unlock()

		if shouldUpload {
			*sendChanges = append(*sendChanges, &fileInformation{
				Name:        relativePath,
				Mtime:       roundMtime(stat.ModTime()),
				Size:        stat.Size(),
				IsDirectory: true,
			})
		}
	}

	for _, f := range files {
		if err := s.diffServerClient(path.Join(filepath, f.Name()), sendChanges, downloadChanges, dontSend); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (s *SyncConfig) sendChangesToUpstream(changes []*fileInformation) {
	for j := 0; j < len(changes); j += initialUpstreamBatchSize {
		// Wait till upstream channel is empty
		for len(s.upstream.events) > 0 {
			time.Sleep(time.Second)
		}

		// Now we send them to upstream
		sendBatch := make([]*fileInformation, 0, initialUpstreamBatchSize)
		s.fileIndex.fileMapMutex.Lock()

		for i := j; i < (j+initialUpstreamBatchSize) && i < len(changes); i++ {
			if s.fileIndex.fileMap[changes[i].Name] == nil || (s.fileIndex.fileMap[changes[i].Name] != nil && changes[i].Mtime > s.fileIndex.fileMap[changes[i].Name].Mtime) {
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
func (s *SyncConfig) Stop(fatalError error) {
	s.stopOnce.Do(func() {
		if s.upstream != nil && s.upstream.interrupt != nil {
			for _, symlink := range s.upstream.symlinks {
				symlink.Stop()
			}

			close(s.upstream.interrupt)

			if s.upstream.stdinPipe != nil {
				s.upstream.stdinPipe.Write([]byte("exit\n"))
				s.upstream.stdinPipe.Close()
			}

			if s.upstream.stdoutPipe != nil {
				s.upstream.stdoutPipe.Close()
			}

			if s.upstream.stderrPipe != nil {
				s.upstream.stderrPipe.Close()
			}
		}

		if s.downstream != nil && s.downstream.interrupt != nil {
			close(s.downstream.interrupt)

			if s.downstream.stdinPipe != nil {
				s.downstream.stdinPipe.Write([]byte("exit\n"))
				s.downstream.stdinPipe.Close()
			}

			if s.downstream.stdoutPipe != nil {
				s.downstream.stdoutPipe.Close()
			}

			if s.downstream.stderrPipe != nil {
				s.downstream.stderrPipe.Close()
			}
		}

		s.Logln("[Sync] Sync stopped")

		if fatalError != nil {
			s.Error(fatalError)
			log.Fatalf("[Sync] Fatal sync error: %v. For more information check .devspace/logs/sync.log", fatalError)
		}
	})
}
