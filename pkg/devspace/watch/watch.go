package watch

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// Callback is the function type
type Callback func(changed []string, deleted []string) error

// Watcher is the struct that contains the watching information
type Watcher struct {
	Paths        []string
	PollInterval time.Duration
	FileMap      map[string]os.FileInfo
	Callback     Callback
	Log          log.Logger

	startOnce sync.Once
	closeOnce sync.Once

	interrupt chan bool
}

// New watches a given glob paths array for changes
func New(paths []string, callback Callback, log log.Logger) (*Watcher, error) {
	watcher := &Watcher{
		Paths:        paths,
		PollInterval: time.Second,
		Callback:     callback,
		FileMap:      make(map[string]os.FileInfo),
		Log:          log,
		interrupt:    make(chan bool),
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
	w.startOnce.Do(func() {
		go func() {
			for {
				select {
				case <-w.interrupt:
					return
				case <-time.After(w.PollInterval):
					changed, deleted, err := w.Update()
					if err != nil {
						w.Log.Errorf("Error during watcher update: %v", err)
						return
					}

					if len(changed) > 0 || len(deleted) > 0 {
						err = w.Callback(changed, deleted)
						if err != nil {
							w.Log.Errorf("Error during watcher callback: %v", err)
							return
						}
					}
				}
			}
		}()
	})
}

// Stop stopps the watcher
func (w *Watcher) Stop() {
	w.closeOnce.Do(func() {
		close(w.interrupt)
	})
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

	// Update map
	w.FileMap = fileMap

	return changed, deleted, nil
}

func (w *Watcher) gatherChanges(newState map[string]os.FileInfo) ([]string, []string) {
	changed := make([]string, 0, 1)
	deleted := make([]string, 0, 1)

	// Get changed paths
	for file, fileInfo := range newState {
		if oldFileInfo, ok := w.FileMap[file]; !ok || oldFileInfo.Size() != fileInfo.Size() || oldFileInfo.ModTime().UnixNano() != fileInfo.ModTime().UnixNano() {
			if strings.HasPrefix(file, ".devspace") {
				continue
			}

			changed = append(changed, file)
		}
	}

	// Get deleted paths
	for file := range w.FileMap {
		if _, ok := newState[file]; !ok {
			if strings.HasPrefix(file, ".devspace") {
				continue
			}

			deleted = append(deleted, file)
		}
	}

	return changed, deleted
}
