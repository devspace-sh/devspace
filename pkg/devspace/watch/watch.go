package watch

import (
	"os"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar"
	"github.com/covexo/devspace/pkg/util/log"
)

// Callback is the function type
type Callback func(changed []string, deleted []string) error

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
		stopChan: make(chan bool),
	}

	// Initialize filemap
	_, _, err := watcher.Update()
	if err != nil {
		return nil, err
	}

	return watcher, nil
}

// Start starts the watching process every second
func (w *Watcher) Start() {
	w.startedMutext.Lock()
	isStarted := w.started
	w.startedMutext.Unlock()

	if isStarted {
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
				changed, deleted, err := w.Update()
				if err != nil {
					w.Log.Errorf("Error during watcher update: %v", err)
					break Outer
				}

				if len(changed) > 0 || len(deleted) > 0 {
					err = w.Callback(changed, deleted)
					if err != nil {
						w.Log.Errorf("Error during watcher callback: %v", err)
						break Outer
					}
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
	w.startedMutext.Lock()
	isStarted := w.started
	w.startedMutext.Unlock()

	if isStarted {
		w.stopChan <- true
	}
}

// Update updates the filemap and returns if there was a change
func (w *Watcher) Update() ([]string, []string, error) {
	fileMap := make(map[string]os.FileInfo)

	for _, pattern := range w.Paths {
		files, err := doublestar.Glob(pattern)
		if err != nil {
			return nil, nil, err
		}

		for _, file := range files {
			stat, err := os.Stat(file)
			if err != nil {
				continue
			}

			fileMap[file] = stat
		}
	}

	changed, deleted := w.gatherChanges(fileMap)
	if len(changed) > 0 || len(deleted) > 0 {
		w.FileMap = fileMap
	}

	return changed, deleted, nil
}

func (w *Watcher) gatherChanges(newState map[string]os.FileInfo) ([]string, []string) {
	changed := make([]string, 0, 1)
	deleted := make([]string, 0, 1)

	// Get changed paths
	for file, fileInfo := range newState {
		if oldFileInfo, ok := w.FileMap[file]; !ok || oldFileInfo.Size() != fileInfo.Size() || oldFileInfo.ModTime().UnixNano() != fileInfo.ModTime().UnixNano() {
			changed = append(changed, file)
		}
	}

	// Get deleted paths
	for file := range w.FileMap {
		if _, ok := newState[file]; !ok {
			deleted = append(deleted, file)
		}
	}

	return changed, deleted
}
