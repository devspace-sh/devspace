package sync

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/juju/errors"
	"github.com/rjeczalik/notify"
	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/sirupsen/logrus"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

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

	fileIndex *fileIndex

	ignoreMatcher         gitignore.IgnoreParser
	downloadIgnoreMatcher gitignore.IgnoreParser
	uploadIgnoreMatcher   gitignore.IgnoreParser

	log *logrus.Logger

	upstream   *upstream
	downstream *downstream

	// Used for testing
	testing   bool
	errorChan chan error
}

// Logf prints the given information to the synclog with context data
func (s *SyncConfig) Logf(format string, args ...interface{}) {
	syncLog.WithKey("local", s.WatchPath).WithKey("container", s.DestPath).WithKey("excluded", s.ExcludePaths).Infof(format, args...)
}

// Logln prints the given information to the synclog with context data
func (s *SyncConfig) Logln(line interface{}) {
	syncLog.WithKey("local", s.WatchPath).WithKey("container", s.DestPath).WithKey("excluded", s.ExcludePaths).Info(line)
}

// Error handles a sync error with context
func (s *SyncConfig) Error(err error) {
	syncLog.WithKey("local", s.WatchPath).WithKey("container", s.DestPath).WithKey("excluded", s.ExcludePaths).Errorf("Error: %v, Stack: %v", err, errors.ErrorStack(err))

	if s.errorChan != nil {
		s.errorChan <- err
	}
}

func (s *SyncConfig) setup() error {
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

	err := s.initIgnoreParsers()
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
		s.Stop()
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
	s.Logf("[Sync] Start syncing\n")

	err := s.initialSync()
	if err != nil {
		s.Error(err)
		return
	}

	// Start upstream
	go func() {
		defer s.Stop()

		// Set up a watchpoint listening for events within a directory tree rooted at specified directory
		err := notify.Watch(s.WatchPath+"/...", s.upstream.events, notify.All)

		if err != nil {
			s.Error(err)
			return
		}

		defer notify.Stop(s.upstream.events)

		err = s.upstream.mainLoop()
		if err != nil {
			s.Error(err)
		}
	}()

	// Start downstream
	go func() {
		defer s.Stop()

		err := s.downstream.mainLoop()
		if err != nil {
			s.Error(err)
		}
	}()
}

func (s *SyncConfig) initialSync() error {
	err := s.downstream.populateFileMap()
	if err != nil {
		return errors.Trace(err)
	}

	remoteChanges := make([]*fileInformation, 0, 10)
	fileMapClone := make(map[string]*fileInformation)

	for key, element := range s.fileIndex.fileMap {
		if element.IsSymbolicLink {
			continue
		}

		fileMapClone[key] = element
	}

	err = s.diffServerClient(s.WatchPath, &remoteChanges, fileMapClone)
	if err != nil {
		return errors.Trace(err)
	}

	if len(remoteChanges) > 0 {
		err = s.upstream.applyCreates(remoteChanges)
		if err != nil {
			return errors.Trace(err)
		}
	}

	if len(fileMapClone) > 0 {
		localChanges := make([]*fileInformation, 0, len(fileMapClone))
		for _, element := range fileMapClone {
			localChanges = append(localChanges, element)
		}

		err = s.downstream.applyChanges(localChanges, nil)
		if err != nil {
			return errors.Trace(err)
		}
	}

	s.Logf("[Sync] Initial sync completed. Processed %d changes", len(remoteChanges)+len(fileMapClone))
	return nil
}

func (s *SyncConfig) diffServerClient(filepath string, sendChanges *[]*fileInformation, downloadChanges map[string]*fileInformation) error {
	relativePath := getRelativeFromFullPath(filepath, s.WatchPath)
	stat, err := os.Lstat(filepath)

	// We skip files that are suddenly not there anymore
	if err != nil {
		return nil
	}

	// We skip symlinks
	if stat.Mode()&os.ModeSymlink != 0 {
		return nil
	}

	// Exclude files on the exclude list
	if s.ignoreMatcher != nil {
		if s.ignoreMatcher.MatchesPath(relativePath) {
			return nil
		}
	}

	delete(downloadChanges, relativePath)

	// Exclude files on the exclude list
	if s.uploadIgnoreMatcher != nil {
		if s.uploadIgnoreMatcher.MatchesPath(relativePath) {
			return nil
		}
	}

	// Exclude remote symlinks
	if s.fileIndex.fileMap[relativePath] != nil && s.fileIndex.fileMap[relativePath].IsSymbolicLink {
		return nil
	}

	if stat.IsDir() {
		return s.diffDir(filepath, sendChanges, downloadChanges)
	}

	// TODO: Handle the case when local files are older than in the container
	if s.fileIndex.fileMap[relativePath] == nil || ceilMtime(stat.ModTime()) > s.fileIndex.fileMap[relativePath].Mtime+1 {
		*sendChanges = append(*sendChanges, &fileInformation{
			Name:        relativePath,
			Mtime:       ceilMtime(stat.ModTime()),
			Size:        stat.Size(),
			IsDirectory: false,
		})
	}

	return nil
}

func (s *SyncConfig) diffDir(filepath string, sendChanges *[]*fileInformation, downloadChanges map[string]*fileInformation) error {
	relativePath := getRelativeFromFullPath(filepath, s.WatchPath)
	files, err := ioutil.ReadDir(filepath)

	if err != nil {
		s.Logf("[Upstream] Couldn't read dir %s: %v", filepath, err)
		return nil
	}

	if len(files) == 0 {
		if s.fileIndex.fileMap[relativePath] == nil {
			*sendChanges = append(*sendChanges, &fileInformation{
				Name:        relativePath,
				IsDirectory: true,
			})
		}
	}

	for _, f := range files {
		if err := s.diffServerClient(path.Join(filepath, f.Name()), sendChanges, downloadChanges); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

//Stop stops the sync process
func (s *SyncConfig) Stop() {
	if s.upstream != nil {
		select {
		case <-s.upstream.interrupt:
		default:
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
	}

	if s.downstream != nil {
		select {
		case <-s.downstream.interrupt:
		default:
			if s.downstream.interrupt != nil {
				close(s.downstream.interrupt)
			}

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
	}
}
