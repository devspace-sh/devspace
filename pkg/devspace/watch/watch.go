package watch

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/sync/util"
	"github.com/pkg/errors"
	gitignore "github.com/sabhiram/go-gitignore"
)

// Callback is the function type
type Callback func(changed []string, deleted []string) error

// Watcher watches a folder
type Watcher interface{
	Start()
	Stop()
	Update() ([]string, []string, error)
}

// watcher is the struct that contains the watching information
type watcher struct {
	Paths   []string
	Exclude gitignore.IgnoreParser

	PollInterval time.Duration
	FileMap      map[string]os.FileInfo
	Callback     Callback
	Log          log.Logger

	startOnce sync.Once
	closeOnce sync.Once

	interrupt chan bool
}

// New watches a given glob paths array for changes
func New(paths []string, exclude []string, pollInterval time.Duration, callback Callback, log log.Logger) (Watcher, error) {
	ignoreMatcher, err := compilePaths(exclude)
	if err != nil {
		return nil, err
	}

	watcher := &watcher{
		Paths:        paths,
		Exclude:      ignoreMatcher,
		PollInterval: pollInterval,
		Callback:     callback,
		FileMap:      make(map[string]os.FileInfo),
		Log:          log,
		interrupt:    make(chan bool),
	}

	// Initialize filemap
	_, _, err = watcher.Update()
	if err != nil {
		return nil, err
	}

	return watcher, nil
}

// Start starts the watching process every second
func (w *watcher) Start() {
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
func (w *watcher) Stop() {
	w.closeOnce.Do(func() {
		close(w.interrupt)
	})
}

// Update updates the filemap and returns if there was a change
func (w *watcher) Update() ([]string, []string, error) {
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

			if w.Exclude != nil && util.MatchesPath(w.Exclude, file, stat.IsDir()) {
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

func (w *watcher) gatherChanges(newState map[string]os.FileInfo) ([]string, []string) {
	changed := make([]string, 0, 1)
	deleted := make([]string, 0, 1)

	// Get changed paths
	for file, fileInfo := range newState {
		oldFileInfo, ok := w.FileMap[file]

		// If existed before
		if ok && oldFileInfo.IsDir() == fileInfo.IsDir() {
			// If directory or file with same size and modification date
			if oldFileInfo.IsDir() || (oldFileInfo.Size() == fileInfo.Size() && oldFileInfo.ModTime().UnixNano() == fileInfo.ModTime().UnixNano()) {
				continue
			}
		} else if strings.HasPrefix(file, ".devspace") {
			continue
		}

		changed = append(changed, file)
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

// compilePaths compiles the exclude paths into an ignore parser
func compilePaths(excludePaths []string) (gitignore.IgnoreParser, error) {
	if len(excludePaths) > 0 {
		ignoreParser, err := gitignore.CompileIgnoreLines(excludePaths...)
		if err != nil {
			return nil, errors.Wrap(err, "compile ignore lines")
		}

		return ignoreParser, nil
	}

	return nil, nil
}
