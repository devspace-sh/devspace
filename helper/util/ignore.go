package util

import (
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

// MatchesPath evaluates if the given relative path matches the matcher
func MatchesPath(matcher gitignore.IgnoreParser, relativePath string, isDir bool) bool {
	relativePath = strings.TrimRight(relativePath, "/")
	if isDir {
		relativePath = relativePath + "/"
	}

	return matcher.MatchesPath(relativePath)
}
