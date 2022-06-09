package sync

import (
	"path/filepath"
	"strings"
)

func getRelativeFromFullPath(fullPath string, prefix string) string {
	return strings.TrimPrefix(strings.ReplaceAll(filepath.ToSlash(fullPath[len(prefix):]), "//", "/"), ".")
}
