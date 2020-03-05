package sync

import (
	"strings"
	"sync"
)

type fileIndex struct {
	fileMap      map[string]*FileInformation
	fileMapMutex sync.Mutex
}

func newFileIndex() *fileIndex {
	return &fileIndex{
		fileMap: make(map[string]*FileInformation),
	}
}

func (f *fileIndex) Lock() {
	f.fileMapMutex.Lock()
}

func (f *fileIndex) Unlock() {
	f.fileMapMutex.Unlock()
}

func (f *fileIndex) Set(file *FileInformation) {
	f.fileMap[file.Name] = file
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
			f.fileMap[subPath] = &FileInformation{
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
