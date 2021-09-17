//go:build linux || darwin
// +build linux darwin

package server

import (
	"os"
	"syscall"
)

// Chown sets the given stat owner and group id to the filepath
func Chown(filepath string, stat os.FileInfo) error {
	// Set old owner & group correctly
	if _, ok := stat.Sys().(*syscall.Stat_t); ok {
		return os.Chown(filepath, int(stat.Sys().(*syscall.Stat_t).Uid), int(stat.Sys().(*syscall.Stat_t).Gid))
	}

	return nil
}
