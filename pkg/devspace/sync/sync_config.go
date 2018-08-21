package sync

import (
	"bytes"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/covexo/devspace/pkg/util/logutil"

	"github.com/Sirupsen/logrus"
	"github.com/rjeczalik/notify"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"github.com/juju/errors"
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
	ExcludeRegEx []string

	fileMap      map[string]*FileInformation
	fileMapMutex sync.Mutex

	compExcludeRegEx []*regexp.Regexp
	log              *logrus.Logger

	upstream   *upstream
	downstream *downstream
}

type FileInformation struct {
	Name        string // %n
	Size        int64  // %s
	Mtime       int64  // %Y
	IsDirectory bool   // parseHex(%f) & S_IFDIR
}

func (s *SyncConfig) Logf(format string, args ...interface{}) {
	syncLog.WithFields(logrus.Fields{
		"PodName":      s.Pod.Name,
		"PodNamespace": s.Pod.Namespace,
		"WatchPath":    s.WatchPath,
		"DestPath":     s.DestPath,
		"ExcludeRegEx": s.ExcludeRegEx,
	}).Infof(format, args)
}

func (s *SyncConfig) Logln(line interface{}) {
	syncLog.WithFields(logrus.Fields{
		"PodName":      s.Pod.Name,
		"PodNamespace": s.Pod.Namespace,
		"WatchPath":    s.WatchPath,
		"DestPath":     s.DestPath,
		"ExcludeRegEx": s.ExcludeRegEx,
	}).Infoln(line)
}

// CopyToContainer copies a local folder or file to a container path
func CopyToContainer(Kubectl *kubernetes.Clientset, Pod *k8sv1.Pod, Container *k8sv1.Container, LocalPath, ContainerPath string, ExcludeRegEx []string) error {
	s := &SyncConfig{
		Kubectl:      Kubectl,
		Pod:          Pod,
		Container:    Container,
		WatchPath:    path.Dir(strings.Replace(LocalPath, "\\", "/", -1)),
		DestPath:     ContainerPath,
		ExcludeRegEx: ExcludeRegEx,
	}

	syncLog = logrus.New()
	syncLog.SetLevel(logrus.InfoLevel)

	if s.ExcludeRegEx != nil {
		compRegExp, err := compileRegExp(s.ExcludeRegEx)

		if err != nil {
			log.Fatal(err)
		}

		s.compExcludeRegEx = compRegExp
	}

	s.fileMap = make(map[string]*FileInformation)
	s.upstream = &upstream{
		config: s,
	}

	err := s.upstream.start()

	if err != nil {
		return err
	}

	stat, err := os.Stat(LocalPath)

	if err != nil {
		return err
	}

	err = s.upstream.sendFiles([]*FileInformation{
		&FileInformation{
			Name:        getRelativeFromFullPath(LocalPath, s.WatchPath),
			IsDirectory: stat.IsDir(),
		},
	})

	if err != nil {
		return err
	}

	s.Stop()
	syncLog = nil

	return nil
}

// Starts a new sync instance
func (s *SyncConfig) Start() {
	if s.ExcludeRegEx == nil {
		s.ExcludeRegEx = make([]string, 0, 2)
	}

	// We exclude the sync log to prevent an endless loop in upstream
	s.ExcludeRegEx = append(s.ExcludeRegEx, "^/\\.devspace\\/logs\\/.*$")

	if syncLog == nil {
		syncLog = logutil.GetLogger("sync", false)

		syncLog.SetLevel(logrus.InfoLevel)
	}

	if s.ExcludeRegEx != nil {
		compRegExp, err := compileRegExp(s.ExcludeRegEx)

		if err != nil {
			log.Fatal(err)
		}

		s.compExcludeRegEx = compRegExp
	}

	s.fileMap = make(map[string]*FileInformation)

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

func compileRegExp(excludeRegEx []string) ([]*regexp.Regexp, error) {
	compExcludeRegEx := make([]*regexp.Regexp, len(excludeRegEx))

	for index, element := range excludeRegEx {
		compiledRegEx, err := regexp.Compile(element)

		if err != nil {
			return nil, err
		}

		compExcludeRegEx[index] = compiledRegEx
	}

	return compExcludeRegEx, nil
}

func (s *SyncConfig) mainLoop() {
	s.Logf("[Sync] Start syncing\n")

	defer s.Stop()
	err := s.downstream.populateFileMap()

	if err != nil {
		syncLog.Errorln(err)
		return
	}

	sendChanges := make([]*FileInformation, 0, 10)
	fileMapClone := make(map[string]*FileInformation)

	for key, element := range s.fileMap {
		fileMapClone[key] = element
	}

	err = s.upstream.diffServerClient(s.WatchPath, &sendChanges, fileMapClone)

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
		downloadChanges := make([]*FileInformation, 0, len(fileMapClone))

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

// Function assumes that fileMap is locked for access
func (s *SyncConfig) createDirInFileMap(dirpath string) {
	if dirpath == "/" {
		return
	}

	pathParts := strings.Split(dirpath, "/")

	for i := len(pathParts); i > 1; i-- {
		subPath := strings.Join(pathParts[:i], "/")

		if s.fileMap[subPath] == nil && subPath != "" {
			s.fileMap[subPath] = &FileInformation{
				Name:        subPath,
				IsDirectory: true,
			}
		}
	}
}

// Function assumes that fileMap is locked for access
// TODO: This function is very expensive O(n), is there a better solution?
func (s *SyncConfig) removeDirInFileMap(dirpath string) {
	if s.fileMap[dirpath] != nil {
		delete(s.fileMap, dirpath)

		dirpath = dirpath + "/"

		for key := range s.fileMap {
			if strings.Index(key, dirpath) == 0 {
				delete(s.fileMap, key)
			}
		}
	}
}

<<<<<<< HEAD
// CopyToContainer copies a local folder or file to a container path
func CopyToContainer(Kubectl *kubernetes.Clientset, Pod *k8sv1.Pod, Container *k8sv1.Container, LocalPath, ContainerPath string) error {
	syncObj := &SyncConfig{
		Kubectl:   Kubectl,
		Pod:       Pod,
		Container: Container,
		WatchPath: path.Dir(strings.Replace(LocalPath, "\\", "/", -1)),
		DestPath:  ContainerPath,
	}

	syncLog = logrus.New()
	syncLog.SetLevel(logrus.InfoLevel)

	syncObj.fileMap = make(map[string]*FileInformation)
	syncObj.upstream = &upstream{
		config: syncObj,
	}

	err := syncObj.upstream.start()

	if err != nil {
		return errors.Trace(err)
	}

	stat, err := os.Stat(LocalPath)

	if err != nil {
		return errors.Trace(err)
	}

	err = syncObj.upstream.sendFiles([]*FileInformation{
		&FileInformation{
			Name:        getRelativeFromFullPath(LocalPath, syncObj.WatchPath),
			IsDirectory: stat.IsDir(),
		},
	})

	if err != nil {
		return errors.Trace(err)
	}

	syncObj.Stop()
	syncLog = nil

	return nil
}

=======
>>>>>>> 69689ba6b9ff0c5edcb07b0faad3540b0f515c76
// We need this function because tar ceils up the mtime to seconds on the server
func ceilMtime(mtime time.Time) int64 {
	if mtime.UnixNano()%1000000000 != 0 {
		num := strconv.FormatInt(mtime.UnixNano(), 10)
		ret, _ := strconv.Atoi(num[:len(num)-9])

		return int64(ret) + 1
	} else {
		return mtime.Unix()
	}
}

func getRelativeFromFullPath(fullpath string, prefix string) string {
	return strings.Replace(strings.Replace(fullpath[len(prefix):], "\\", "/", -1), "//", "/", -1)
}

func pipeStream(w io.Writer, r io.Reader) error {
	buf := make([]byte, 1024, 1024)

	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]

			_, err := w.Write(d)
			if err != nil {
				return errors.Trace(err)
			}
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}
			return errors.Trace(err)
		}
	}
}

func readTill(keyword string, reader io.Reader) (string, error) {
	var output bytes.Buffer
	buf := make([]byte, 0, 512)
	overlap := ""

	for keywordFound := false; keywordFound == false; {
		n, err := reader.Read(buf[:cap(buf)])

		buf = buf[:n]

		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}

			return "", errors.Trace(err)
		}

		// process buf
		if err != nil && err != io.EOF {
			return "", errors.Trace(err)
		}

		lines := strings.Split(string(buf), "\n")

		for index, element := range lines {
			line := ""

			if index == 0 {
				if len(lines) > 1 {
					line = overlap + element
				} else {
					overlap += element
				}
			} else if index == len(lines)-1 {
				overlap = element
			} else {
				line = element
			}

			if line == keyword {
				output.WriteString(line)
				keywordFound = true
				break
			} else if overlap == keyword {
				output.WriteString(overlap)
				keywordFound = true
				break
			} else if line != "" {
				output.WriteString(line + "\n")
			}
		}
	}

	return output.String(), nil
}

func waitTill(keyword string, reader io.Reader) error {
	buf := make([]byte, 0, 512)
	overlap := ""

	for {
		n, err := reader.Read(buf[:cap(buf)])

		buf = buf[:n]

		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}

			return errors.Trace(err)
		}

		// process buf
		if err != nil && err != io.EOF {
			return errors.Trace(err)
		}

		lines := strings.Split(string(buf), "\n")

		for index, element := range lines {
			line := ""

			if index == 0 {
				if len(lines) > 1 {
					line = overlap + element
				} else {
					overlap += element
				}
			} else if index == len(lines)-1 {
				overlap = element
			} else {
				line = element
			}

			if line == keyword || overlap == keyword {
				return nil
			}
		}
	}

	return nil
}

// clean prevents path traversals by stripping them out.
// This is adapted from https://golang.org/src/net/http/fs.go#L74
func clean(fileName string) string {
	return path.Clean(string(os.PathSeparator) + fileName)
}

// dirExists checks if a path exists and is a directory.
func dirExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err == nil && fi.IsDir() {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, errors.Trace(err)
}

func deleteSafe(path string, fileInformation *FileInformation) bool {
	// Only delete if mtime and size did not change
	stat, err := os.Stat(path)

	if err == nil && stat.Size() == fileInformation.Size && ceilMtime(stat.ModTime()) == fileInformation.Mtime {
		err = os.Remove(path)

		if err == nil {
			return true
		}
	}

	return false
}
