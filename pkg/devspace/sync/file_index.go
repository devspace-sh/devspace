package sync

import (
	"strings"
	"sync"
)

type fileIndex struct {
	fileMap      map[string]*fileInformation
	fileMapMutex sync.Mutex
}

func newFileIndex() *fileIndex {
	return &fileIndex{
		fileMap: make(map[string]*fileInformation),
	}
}

func (f *fileIndex) ExecuteSafeError(fn func(fileMap map[string]*fileInformation) error) error {
	f.fileMapMutex.Lock()
	defer f.fileMapMutex.Unlock()

	return fn(f.fileMap)
}

func (f *fileIndex) ExecuteSafe(fn func(fileMap map[string]*fileInformation)) {
	f.fileMapMutex.Lock()
	defer f.fileMapMutex.Unlock()

	fn(f.fileMap)
}

// Function assumes that fileMap is locked for access
func (f *fileIndex) CreateDirInFileMap(dirpath string) {
	if dirpath == "/" {
		return
	}

	pathParts := strings.Split(dirpath, "/")

	for i := len(pathParts); i > 1; i-- {
		subPath := strings.Join(pathParts[:i], "/")

		if f.fileMap[subPath] == nil && subPath != "" {
			f.fileMap[subPath] = &fileInformation{
				Name:        subPath,
				IsDirectory: true,
			}
		}
	}
}

// Function assumes that fileMap is locked for access
// TODO: This function is very expensive O(n), is there a better solution?
func (f *fileIndex) RemoveDirInFileMap(dirpath string) {
	if f.fileMap[dirpath] != nil {
		delete(f.fileMap, dirpath)

		dirpath = dirpath + "/"

		for key := range f.fileMap {
			if strings.Index(key, dirpath) == 0 {
				delete(f.fileMap, key)
			}
		}
	}
}
