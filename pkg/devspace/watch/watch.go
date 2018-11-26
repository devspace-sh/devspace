package watch

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/covexo/devspace/pkg/util/log"
)

// Callback is the function type
type Callback func() error

// Watcher is the struct that contains the watching information
type Watcher struct {
	Paths    []string
	FileMap  map[string]os.FileInfo
	Callback Callback
	Log      log.Logger

	started       bool
	startedMutext sync.Mutex
	stopChan      chan bool
}

// New watches a given glob paths array for changes
func New(paths []string, callback Callback, log log.Logger) (*Watcher, error) {
	watcher := &Watcher{
		Paths:    paths,
		Callback: callback,
		FileMap:  make(map[string]os.FileInfo),
		Log:      log,
	}

	_, err := watcher.Update()
	if err != nil {
		return nil, err
	}

	return watcher, nil
}

// Start starts the watching process every second
func (w *Watcher) Start() {
	if w.started {
		return
	}

	w.startedMutext.Lock()
	w.started = true
	w.startedMutext.Unlock()

	go func() {
	Outer:
		for {
			select {
			case <-w.stopChan:
				break Outer
			case <-time.After(time.Second):
				changed, err := w.Update()
				if err != nil {
					w.Log.Errorf("Error during watcher update: %v", err)
					break Outer
				}

				if changed {
					err = w.Callback()
					if err != nil {
						w.Log.Errorf("Error during watcher callback: %v", err)
						break Outer
					}

					break Outer
				}
			}
		}

		w.startedMutext.Lock()
		w.started = false
		w.startedMutext.Unlock()
	}()
}

// Stop stopps the watcher
func (w *Watcher) Stop() {
	if w.started {
		w.stopChan <- true
	}
}

// Update updates the filemap and returns if there was a change
func (w *Watcher) Update() (bool, error) {
	fileMap := make(map[string]os.FileInfo)

	for _, pattern := range w.Paths {
		files, err := filepath.Glob(pattern)
		if err != nil {
			return false, err
		}

		for _, file := range files {
			stat, err := os.Stat(file)
			if err != nil {
				continue
			}

			fileMap[file] = stat
		}
	}

	changed := w.hasChanged(fileMap)
	if changed {
		w.FileMap = fileMap
	}

	return changed, nil
}

func (w *Watcher) hasChanged(newState map[string]os.FileInfo) bool {
	if len(w.FileMap) != len(newState) {
		return true
	}

	for file, fileInfo := range w.FileMap {
		if newFileInfo, ok := newState[file]; !ok || newFileInfo.Size() != fileInfo.Size() || newFileInfo.ModTime().UnixNano() != fileInfo.ModTime().UnixNano() {
			return true
		}
	}

	return false
}
