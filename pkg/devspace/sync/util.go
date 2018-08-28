package sync

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/juju/errors"
	gitignore "github.com/sabhiram/go-gitignore"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

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

	syncLog = log.GetInstance()

	if s.ExcludePaths != nil {
		ignoreMatcher, err := compilePaths(s.ExcludePaths)

		if err != nil {
			return errors.Trace(err)
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

func deleteSafeRecursive(basepath, relativePath string, fileMap map[string]*fileInformation, removeFiles map[string]*fileInformation, config *SyncConfig) {
	absolutePath := path.Join(basepath, relativePath)
	relativePath = getRelativeFromFullPath(absolutePath, basepath)

	// We don't delete the folder or the contents if we haven't tracked it
	if fileMap[relativePath] == nil || removeFiles[relativePath] == nil {
		config.Logf("[Downstream] Skip delete directory %s\n", relativePath)

		return
	}

	// Delete directory from fileMap
	defer delete(fileMap, relativePath)
	files, err := ioutil.ReadDir(absolutePath)

	if err != nil {
		return
	}

	for _, f := range files {
		if f.IsDir() {
			deleteSafeRecursive(basepath, path.Join(relativePath, f.Name()), fileMap, removeFiles, config)
		} else {
			filepath := path.Join(relativePath, f.Name())
			fileDeleted := false

			// We don't delete the file if we haven't tracked it
			if fileMap[filepath] != nil && removeFiles[filepath] != nil {
				// We don't delete the file if it has changed in the map since we collected changes
				if removeFiles[filepath].Mtime == fileMap[filepath].Mtime && removeFiles[filepath].Size == fileMap[filepath].Size {
					// We don't delete the file if it has changed on the filesystem meanwhile
					fileDeleted = deleteSafe(path.Join(basepath, filepath), fileMap[filepath])
				}
			}

			if fileDeleted == false {
				config.Logf("[Downstream] Skip file delete %s\n", relativePath)
			} else {
				delete(fileMap, filepath)
			}
		}
	}

	// This will not remove the directory if there is still a file or directory in it
	err = os.Remove(absolutePath)

	if err != nil {
		config.Logf("[Downstream] Skip delete directory %s, because %s\n", relativePath, err.Error())
	}
}

func deleteSafe(path string, fileInformation *fileInformation) bool {
	// Only delete if mtime and size did not change
	stat, err := os.Stat(path)

	// TODO: uncomment this line for more safety (However we have to change the intial sync functionality that older files locally are either uplaoded or the newer files on the server downloaded)
	// if err == nil && stat.Size() == fileInformation.Size && ceilMtime(stat.ModTime()) == fileInformation.Mtime {
	if err == nil && ceilMtime(stat.ModTime()) <= fileInformation.Mtime {
		err = os.Remove(path)

		if err == nil {
			return true
		}
	}

	return false
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
