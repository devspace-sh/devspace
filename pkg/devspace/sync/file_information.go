package sync

import (
	"os"

	"github.com/loft-sh/devspace/helper/remote"
	"github.com/loft-sh/notify"
)

// FileInformation describes a path or file that is synced either in the remote container or locally
type FileInformation struct {
	Name           string
	Size           int64
	Mtime          int64
	MtimeNano      int64
	Mode           os.FileMode
	IsDirectory    bool
	IsSymbolicLink bool
	ResolvedLink   bool
	Files          int
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

func parseFileInformation(change *remote.Change) *FileInformation {
	return &FileInformation{
		Name:        change.Path,
		Size:        change.Size,
		Mtime:       change.MtimeUnix,
		MtimeNano:   change.MtimeUnixNano,
		Mode:        os.FileMode(change.Mode),
		IsDirectory: change.IsDir,
	}
}
