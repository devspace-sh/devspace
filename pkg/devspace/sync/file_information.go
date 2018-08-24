package sync

import (
	"strconv"
	"strings"

	"github.com/juju/errors"
	gitignore "github.com/sabhiram/go-gitignore"
)

type fileInformation struct {
	Name        string // %n
	Size        int64  // %s
	Mtime       int64  // %Y
	IsDirectory bool   // parseHex(%f) & S_IFDIR
}

func parseFileInformation(fileline, destPath string, ignoreMatcher gitignore.IgnoreParser) (*fileInformation, error) {
	fileinfo := fileInformation{}

	t := strings.Split(fileline, "///")

	if len(t) != 2 {
		return nil, errors.New("[Downstream] Wrong fileline: " + fileline)
	}

	if len(t[0]) <= len(destPath) {
		return nil, nil
	}

	fileinfo.Name = t[0][len(destPath):]

	if ignoreMatcher != nil {
		if ignoreMatcher.MatchesPath(fileinfo.Name) {
			return nil, nil
		}
	}

	t = strings.Split(t[1], ",")

	if len(t) != 3 {
		return nil, errors.New("[Downstream] Wrong fileline: " + fileline)
	}

	size, err := strconv.Atoi(t[0])

	if err != nil {
		return nil, errors.Trace(err)
	}

	fileinfo.Size = int64(size)

	mTime, err := strconv.Atoi(t[1])

	if err != nil {
		return nil, errors.Trace(err)
	}

	fileinfo.Mtime = int64(mTime)

	rawMode, err := strconv.ParseUint(t[2], 16, 32) // Parse hex string into uint64

	if err != nil {
		return nil, errors.Trace(err)
	}

	// We skip symbolic links for now, because windows has problems with them
	if rawMode&IsSymbolicLink == IsSymbolicLink {
		return nil, nil
	}

	fileinfo.IsDirectory = (rawMode & IsDirectory) == IsDirectory

	return &fileinfo, nil
}
