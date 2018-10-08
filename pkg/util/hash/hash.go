package hash

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// Directory creates the hash value of a directory
func Directory(path string) (string, error) {
	hash := sha256.New()
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// We ignore errors
			return nil
		}

		size := strconv.FormatInt(info.Size(), 10)
		mTime := strconv.FormatInt(info.ModTime().UnixNano(), 10)
		io.WriteString(hash, path+";"+size+";"+mTime)

		return nil
	})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
