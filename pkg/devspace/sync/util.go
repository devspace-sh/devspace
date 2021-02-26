package sync

import (
	"path/filepath"
	"strings"

	"github.com/juju/errors"
	gitignore "github.com/sabhiram/go-gitignore"
)

func getRelativeFromFullPath(fullpath string, prefix string) string {
	return strings.TrimPrefix(strings.Replace(filepath.ToSlash(fullpath[len(prefix):]), "//", "/", -1), ".")
}

// CompilePaths compiles the exclude paths into an ignore parser
func CompilePaths(excludePaths []string) (gitignore.IgnoreParser, error) {
	if len(excludePaths) > 0 {
		ignoreParser, err := gitignore.CompileIgnoreLines(excludePaths...)
		if err != nil {
			return nil, errors.Trace(err)
		}

		return ignoreParser, nil
	}

	return nil, nil
}
