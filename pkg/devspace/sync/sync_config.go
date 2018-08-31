package sync

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/Sirupsen/logrus"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/juju/errors"
	"github.com/rjeczalik/notify"
	gitignore "github.com/sabhiram/go-gitignore"
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
	Kubectl      *kubernetes.Clientset
	Pod          *k8sv1.Pod
	Container    *k8sv1.Container
	WatchPath    string
	DestPath     string
	ExcludePaths []string

	fileIndex *fileIndex

	ignoreMatcher gitignore.IgnoreParser
	log           *logrus.Logger

	upstream   *upstream
	downstream *downstream
}

// Logf prints the given information to the synclog with context data
func (s *SyncConfig) Logf(format string, args ...interface{}) {
	syncLog.WithKey("pod", s.Pod.Name).WithKey("namespace", s.Pod.Namespace).WithKey("local", s.WatchPath).WithKey("container", s.DestPath).WithKey("excluded", s.ExcludePaths).Infof(format, args...)
}

// Logln prints the given information to the synclog with context data
func (s *SyncConfig) Logln(line interface{}) {
	syncLog.WithKey("pod", s.Pod.Name).WithKey("namespace", s.Pod.Namespace).WithKey("local", s.WatchPath).WithKey("container", s.DestPath).WithKey("excluded", s.ExcludePaths).Info(line)
}

// Error handles a sync error with context
func (s *SyncConfig) Error(line interface{}) {
	syncLog.WithKey("pod", s.Pod.Name).WithKey("namespace", s.Pod.Namespace).WithKey("local", s.WatchPath).WithKey("container", s.DestPath).WithKey("excluded", s.ExcludePaths).Error(line)
}

// Start starts a new sync instance
func (s *SyncConfig) Start() error {
	if s.ExcludePaths == nil {
		s.ExcludePaths = make([]string, 0, 2)
	}

	// We exclude the sync log to prevent an endless loop in upstream
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

	if s.ExcludePaths != nil {
		ignoreMatcher, err := compilePaths(s.ExcludePaths)

		if err != nil {
			return err
		}

		s.ignoreMatcher = ignoreMatcher
	}

	s.fileIndex = newFileIndex()
	s.upstream = &upstream{
		config: s,
	}

	err := s.upstream.start()

	if err != nil {
		return err
	}

	s.downstream = &downstream{
		config: s,
	}

	err = s.downstream.start()

	if err != nil {
		s.Stop()

		return err
	}

	go s.mainLoop()

	return nil
}

func (s *SyncConfig) mainLoop() {
	s.Logf("[Sync] Start syncing\n")
	err := s.initialSync()

	if err != nil {
		s.Error(err)
		return
	}

	// Run upstream in goroutine
	go func() {
		defer s.Stop()

		// Set up a watchpoint listening for events within a directory tree rooted at specified directory.
		if err := notify.Watch(s.WatchPath+"/...", s.upstream.events, notify.All); err != nil {
			s.Error(err)
			return
		}

		defer notify.Stop(s.upstream.events)
		err := s.upstream.collectChanges()

		if err != nil {
			s.Error(err)
		}
	}()

	// Run downstream in goroutine
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

	sendChanges := make([]*fileInformation, 0, 10)
	fileMapClone := make(map[string]*fileInformation)

	for key, element := range s.fileIndex.fileMap {
		fileMapClone[key] = element
	}

	err = s.fileIndex.ExecuteSafeError(func(fileMap map[string]*fileInformation) error {
		return s.diffServerClient(s.WatchPath, fileMap, &sendChanges, fileMapClone)
	})

	if err != nil {
		return errors.Trace(err)
	}

	if len(sendChanges) > 0 {
		s.Logf("[Sync] Upload %d changes initially\n", len(sendChanges))
		err = s.upstream.sendFiles(sendChanges)

		if err != nil {
			return errors.Trace(err)
		}
	}

	if len(fileMapClone) > 0 {
		downloadChanges := make([]*fileInformation, 0, len(fileMapClone))

		for _, element := range fileMapClone {
			downloadChanges = append(downloadChanges, element)
		}

		s.Logf("[Sync] Download %d changes initially\n", len(downloadChanges))
		err = s.downstream.applyChanges(downloadChanges, nil)

		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (s *SyncConfig) diffServerClient(filepath string, fileMap map[string]*fileInformation, sendChanges *[]*fileInformation, downloadChanges map[string]*fileInformation) error {
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

	if stat.IsDir() {
		delete(downloadChanges, relativePath)

		files, err := ioutil.ReadDir(filepath)

		if err != nil {
			s.Logf("[Upstream] Couldn't read dir %s: %s\n", filepath, err.Error())
			return nil
		}

		if len(files) == 0 {
			if fileMap[relativePath] == nil {
				*sendChanges = append(*sendChanges, &fileInformation{
					Name:        relativePath,
					IsDirectory: true,
				})
			}
		}

		for _, f := range files {
			if err := s.diffServerClient(path.Join(filepath, f.Name()), fileMap, sendChanges, downloadChanges); err != nil {
				return errors.Trace(err)
			}
		}

		return nil
	}
	delete(downloadChanges, relativePath)

	// TODO: Handle the case when local files are older than in the container
	if fileMap[relativePath] == nil || ceilMtime(stat.ModTime()) > fileMap[relativePath].Mtime+1 {
		*sendChanges = append(*sendChanges, &fileInformation{
			Name:        relativePath,
			Mtime:       ceilMtime(stat.ModTime()),
			Size:        stat.Size(),
			IsDirectory: false,
		})
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
	}
}
