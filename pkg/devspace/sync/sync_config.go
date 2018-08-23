package sync

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/covexo/devspace/pkg/util/logutil"

	"github.com/Sirupsen/logrus"
	"github.com/juju/errors"
	"github.com/rjeczalik/notify"
	gitignore "github.com/sabhiram/go-gitignore"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

var syncLog *logrus.Logger

const StartAck string = "START"
const EndAck string = "DONE"
const ErrorAck string = "ERROR"

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

func (s *SyncConfig) Logf(format string, args ...interface{}) {
	syncLog.WithFields(logrus.Fields{
		"PodName":      s.Pod.Name,
		"PodNamespace": s.Pod.Namespace,
		"WatchPath":    s.WatchPath,
		"DestPath":     s.DestPath,
		"ExcludePaths": s.ExcludePaths,
	}).Infof(format, args)
}

func (s *SyncConfig) Logln(line interface{}) {
	syncLog.WithFields(logrus.Fields{
		"PodName":      s.Pod.Name,
		"PodNamespace": s.Pod.Namespace,
		"WatchPath":    s.WatchPath,
		"DestPath":     s.DestPath,
		"ExcludePaths": s.ExcludePaths,
	}).Infoln(line)
}

// CopyToContainer copies a local folder or file to a container path
func CopyToContainer(Kubectl *kubernetes.Clientset, Pod *k8sv1.Pod, Container *k8sv1.Container, LocalPath, ContainerPath string, ExcludePaths []string) error {
	s := &SyncConfig{
		Kubectl:      Kubectl,
		Pod:          Pod,
		Container:    Container,
		WatchPath:    path.Dir(strings.Replace(LocalPath, "\\", "/", -1)),
		DestPath:     ContainerPath,
		ExcludePaths: ExcludePaths,
	}

	syncLog = logrus.New()
	syncLog.SetLevel(logrus.InfoLevel)

	if s.ExcludePaths != nil {
		ignoreMatcher, err := compilePaths(s.ExcludePaths)

		if err != nil {
			log.Fatal(err)
		}

		s.ignoreMatcher = ignoreMatcher
	}

	s.fileIndex = newFileIndex()
	s.upstream = &upstream{
		config: s,
	}

	err := s.upstream.start()

	if err != nil {
		return errors.Trace(err)
	}

	stat, err := os.Stat(LocalPath)

	if err != nil {
		return errors.Trace(err)
	}

	err = s.upstream.sendFiles([]*fileInformation{
		&fileInformation{
			Name:        getRelativeFromFullPath(LocalPath, s.WatchPath),
			IsDirectory: stat.IsDir(),
		},
	})

	if err != nil {
		return errors.Trace(err)
	}

	s.Stop()
	syncLog = nil

	return nil
}

// Starts a new sync instance
func (s *SyncConfig) Start() {
	if s.ExcludePaths == nil {
		s.ExcludePaths = make([]string, 0, 2)
	}

	// We exclude the sync log to prevent an endless loop in upstream
	s.ExcludePaths = append(s.ExcludePaths, "^/\\.devspace\\/logs\\/.*$")

	if syncLog == nil {
		syncLog = logutil.GetLogger("sync", false)

		syncLog.SetLevel(logrus.InfoLevel)
	}

	if s.ExcludePaths != nil {
		ignoreMatcher, err := compilePaths(s.ExcludePaths)

		if err != nil {
			log.Fatal(err)
		}

		s.ignoreMatcher = ignoreMatcher
	}

	s.fileIndex = newFileIndex()
	s.upstream = &upstream{
		config: s,
	}

	err := s.upstream.start()

	if err != nil {
		log.Fatal(err)
	}

	s.downstream = &downstream{
		config: s,
	}

	err = s.downstream.start()

	if err != nil {
		log.Fatal(err)
	}

	go s.mainLoop()
}

func compilePaths(excludePaths []string) (gitignore.IgnoreParser, error) {
	if len(excludePaths) > 0 {
		ignoreParser, err := gitignore.CompileIgnoreLines(excludePaths...)

		if err != nil {
			return nil, errors.Trace(err)
		}

		return ignoreParser, nil
	}

	return nil, nil
}

func (s *SyncConfig) mainLoop() {
	s.Logf("[Sync] Start syncing\n")

	defer s.Stop()
	err := s.downstream.populateFileMap()

	if err != nil {
		syncLog.Errorln(err)
		return
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
		syncLog.Errorln(err)
		return
	}

	if len(sendChanges) > 0 {
		s.Logf("[Sync] Upload %d changes initially\n", len(sendChanges))
		err = s.upstream.sendFiles(sendChanges)

		if err != nil {
			syncLog.Errorln(err)
			return
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
			syncLog.Errorln(err)
			return
		}
	}

	// Run upstream
	go func() {
		defer s.Stop()

		// Set up a watchpoint listening for events within a directory tree rooted at specified directory.
		if err := notify.Watch(s.WatchPath+"/...", s.upstream.events, notify.All); err != nil {
			syncLog.Errorln(err)
			return
		}

		defer notify.Stop(s.upstream.events)
		err := s.upstream.collectChanges()

		if err != nil {
			syncLog.Errorln(err)
		}
	}()

	err = s.downstream.mainLoop()

	if err != nil {
		syncLog.Errorln(err)
	}
}

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

	delete(downloadChanges, relativePath)

	if stat.IsDir() {
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
	} else {
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
}
