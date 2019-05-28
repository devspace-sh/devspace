package sync

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/util/log"
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

func cleanupSyncLogs() error {
	syncLogName := log.Logdir + "sync.log"
	_, err := os.Stat(syncLogName)
	if err != nil {
		return nil
	}

	// We read the log file and append it to the old log
	data, err := ioutil.ReadFile(syncLogName)
	if err != nil {
		return err
	}

	// Append to syncLog.log.old
	f, err := os.OpenFile(syncLogName+".old", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err = f.Write(data); err != nil {
		return err
	}

	return os.Remove(syncLogName)
}
