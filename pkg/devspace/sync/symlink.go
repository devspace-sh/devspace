package sync

import (
	"os"
	"path/filepath"

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

	events   chan notify.EventInfo
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
	watchPath := targetPath

	if isDir {
		watchPath += "/..."
		symlink.events = make(chan notify.EventInfo, 100)
	} else {
		symlink.events = make(chan notify.EventInfo, 10)
	}

	// Set up a watchpoint listening for events within a directory tree rooted at specified directory
	err := notify.Watch(watchPath, symlink.events, notify.All)
	if err != nil {
		return nil, err
	}

	go symlink.loop()

	return symlink, nil
}

func (s *Symlink) loop() {
	for event := range s.events {
		s.upstream.events <- &symlinkEvent{
			event: event.Event(),
			path:  s.rewritePath(event.Path()),
		}
	}
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
	notify.Stop(s.events)
	close(s.events)
}
