//go:build windows
// +build windows

package server

import "os"

// Chown sets the given stat owner and group id to the filepath
func Chown(filepath string, stat os.FileInfo) error {
	return nil
}
