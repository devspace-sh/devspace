package sync

import (
	"github.com/loft-sh/devspace/helper/remote"
	"github.com/syncthing/notify"
)

// FileInformation describes a path or file that is synced either in the remote container or locally
type FileInformation struct {
	Name      string
	Size      int64
	Mtime     int64
	MtimeNano int64

	IsSymbolicLink bool
	IsDirectory    bool
}

// Sys implements interface
func (f *FileInformation) Sys() interface{} {
	return f
}

// Path implements interface
func (f *FileInformation) Path() string {
	return f.Name
}

// Event implements interface
func (f *FileInformation) Event() notify.Event {
	if f.Mtime == 0 {
		return notify.Remove
	}

	return notify.Create
}

type parsingError struct {
	msg string
}

func (p parsingError) Error() string {
	return p.msg
}

func parseFileInformation(change *remote.Change) *FileInformation {
	return &FileInformation{
		Name:        change.Path,
		Size:        change.Size,
		Mtime:       change.MtimeUnix,
		MtimeNano:   change.MtimeUnixNano,
		IsDirectory: change.IsDir,
	}
}
