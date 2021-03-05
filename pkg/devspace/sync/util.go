package sync

import (
	"path/filepath"
	"strings"
)

func getRelativeFromFullPath(fullpath string, prefix string) string {
	return strings.TrimPrefix(strings.Replace(filepath.ToSlash(fullpath[len(prefix):]), "//", "/", -1), ".")
}
