package watch

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"

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

// GetPaths retrieves the watch paths from the config object
func GetPaths() []string {
	paths := make([]string, 0, 1)
	config := configutil.GetConfig()

	// Add the deploy manifest paths
	if config.DevSpace != nil && config.DevSpace.Deployments != nil {
		for _, deployConf := range *config.DevSpace.Deployments {
			if deployConf.AutoReload != nil && deployConf.AutoReload.Disabled != nil && *deployConf.AutoReload.Disabled == true {
				continue
			}

			if deployConf.Helm != nil && deployConf.Helm.ChartPath != nil {
				chartPath := *deployConf.Helm.ChartPath
				if chartPath[len(chartPath)-1] != '/' {
					chartPath += "/"
				}

				paths = append(paths, chartPath+"**")
			} else if deployConf.Kubectl != nil && deployConf.Kubectl.Manifests != nil {
				for _, manifestPath := range *deployConf.Kubectl.Manifests {
					paths = append(paths, *manifestPath)
				}
			}
		}
	}

	// Add the dockerfile paths
	if config.Images != nil {
		for _, imageConf := range *config.Images {
			if imageConf.AutoReload != nil && imageConf.AutoReload.Disabled != nil && *imageConf.AutoReload.Disabled == true {
				continue
			}

			dockerfilePath := "./Dockerfile"
			if imageConf.Build != nil && imageConf.Build.DockerfilePath != nil {
				dockerfilePath = *imageConf.Build.DockerfilePath
			}

			paths = append(paths, dockerfilePath)
		}
	}

	// Add the additional paths
	if config.DevSpace != nil && config.DevSpace.AutoReload != nil && config.DevSpace.AutoReload.Paths != nil {
		for _, path := range *config.DevSpace.AutoReload.Paths {
			paths = append(paths, *path)
		}
	}

	return paths
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
