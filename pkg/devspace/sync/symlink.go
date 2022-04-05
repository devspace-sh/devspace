package sync

import (
	"os"
	"path/filepath"

	"github.com/loft-sh/devspace/helper/server/ignoreparser"
	"github.com/loft-sh/notify"
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

	events   chan notify.EventInfo
	watcher  notify.Tree
	upstream *upstream
}

// NewSymlink creates a new symlink object
func NewSymlink(upstream *upstream, symlinkPath, targetPath string, isDir bool, ignoreMatcher ignoreparser.IgnoreParser) (*Symlink, error) {
	symlink := &Symlink{
		SymlinkPath: symlinkPath,
		TargetPath:  targetPath,
		IsDir:       isDir,
		events:      make(chan notify.EventInfo, 1000),
		upstream:    upstream,
	}

	symlink.watcher = notify.NewTree()
	_ = symlink.watcher.Watch(targetPath, symlink.events, func(path string) bool {
		if ignoreMatcher == nil || ignoreMatcher.RequireFullScan() {
			return false
		}

		stat, err := os.Stat(path)
		if err != nil {
			return false
		}

		return ignoreMatcher.Matches(path[len(symlink.SymlinkPath):], stat.IsDir())
	}, notify.All)

	go func() {
		for event := range symlink.events {
			symlink.upstream.events <- &symlinkEvent{
				path:  symlink.rewritePath(event.Path()),
				event: event.Event(),
			}
		}
	}()

	return symlink, nil
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
	s.watcher.Stop(s.events)
}
