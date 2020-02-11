package sync

import (
	"os"
	"path/filepath"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/watch"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/rjeczalik/notify"
)

type symlinkEvent struct {
	path  string
	event notify.Event
}

func (s *symlinkEvent) Event() notify.Event {
	return s.event
}
func (s *symlinkEvent) Path() string {
	return s.path
}
func (s *symlinkEvent) Sys() interface{} {
	return nil
}

// Symlink holds information about a symlink
type Symlink struct {
	SymlinkPath string
	TargetPath  string

	IsDir bool

	watcher  watch.Watcher
	upstream *upstream
}

// NewSymlink creates a new symlink object
func NewSymlink(upstream *upstream, symlinkPath, targetPath string, isDir bool) (*Symlink, error) {
	symlink := &Symlink{
		SymlinkPath: symlinkPath,
		TargetPath:  targetPath,
		IsDir:       isDir,
		upstream:    upstream,
	}

	watchPath := filepath.ToSlash(targetPath)
	if isDir {
		watchPath += "/**"
	}

	watcher, err := watch.New([]string{watchPath}, []string{}, time.Millisecond * 500, symlink.handleChange, log.Discard)
	if err != nil {
		return nil, err
	}

	symlink.watcher = watcher
	symlink.watcher.Start()

	return symlink, nil
}

func (s *Symlink) handleChange(changed []string, deleted []string) error {
	for _, path := range changed {
		s.upstream.events <- &symlinkEvent{
			path:  s.rewritePath(path),
			event: notify.Create,
		}
	}

	for _, path := range deleted {
		s.upstream.events <- &symlinkEvent{
			path:  s.rewritePath(path),
			event: notify.Remove,
		}
	}

	return nil
}

func (s *Symlink) rewritePath(path string) string {
	return s.SymlinkPath + path[len(s.TargetPath):]
}

// Crawl resolves the symlink and sends an event for each file in the target path
func (s *Symlink) Crawl() error {
	return filepath.Walk(s.TargetPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		s.upstream.events <- &symlinkEvent{
			event: notify.Create,
			path:  s.rewritePath(path),
		}

		return nil
	})
}

// Stop stops watching on the watching path
func (s *Symlink) Stop() {
	s.watcher.Stop()
}
