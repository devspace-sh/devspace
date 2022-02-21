// Package single provides a mechanism to ensure, that only one instance of a program is running
package single

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	// ErrAlreadyRunning another instance of is already running
	ErrAlreadyRunning = errors.New("another instance is already running")
)

// Single represents the name and the open file descriptor
type Single struct {
	name string
	path string
	file *os.File
}

// New creates a Single instance where name is the basename of the lock file (<name>.lock)
// if no path is given (WithLockPath option) the lock will be created in an operating specific path as <name>.lock
func New(name string, opts ...Option) (*Single, error) {
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	s := &Single{
		name: name,
	}

	for _, o := range opts {
		o(s)
	}

	if s.path == "" {
		s.path = os.TempDir()
	}

	return s, nil
}

// Lockfile returns the full path of the lock file
func (s *Single) Lockfile() string {
	return filepath.Join(s.path, fmt.Sprintf("%s.lock", s.name))
}
