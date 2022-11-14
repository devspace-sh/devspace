package fsutil

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	recursiveCopy "github.com/otiai10/copy"
)

// IsRecursiveSymlink checks if the provided non-resolved file info
// is a recursive symlink
func IsRecursiveSymlink(f os.FileInfo, symlinkPath string) bool {
	// check if recursive symlink
	if f.Mode()&os.ModeSymlink == os.ModeSymlink {
		resolvedPath, err := filepath.EvalSymlinks(symlinkPath)
		if err != nil || strings.HasPrefix(symlinkPath, filepath.ToSlash(resolvedPath)) {
			return true
		}
	}

	return false
}

// WriteToFile writes data to a file
func WriteToFile(data []byte, filePath string) error {
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0666)
}

// ReadFile reads a file with a given limit
func ReadFile(path string, limit int64) ([]byte, error) {
	if limit <= 0 {
		return os.ReadFile(path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}

	size := st.Size()
	if limit > 0 && size > limit {
		size = limit
	}

	buf := bytes.NewBuffer(nil)
	buf.Grow(int(size))
	_, err = io.Copy(buf, io.LimitReader(f, limit))

	return buf.Bytes(), err
}

// Copy copies a file to a destination path
func Copy(sourcePath string, targetPath string, overwrite bool) error {
	if overwrite {
		return recursiveCopy.Copy(sourcePath, targetPath)
	}

	var err error

	// Convert to absolute path
	sourcePath, err = filepath.Abs(sourcePath)
	if err != nil {
		return err
	}

	// Convert to absolute path
	targetPath, err = filepath.Abs(targetPath)
	if err != nil {
		return err
	}

	return filepath.Walk(sourcePath, func(nextSourcePath string, fileInfo os.FileInfo, err error) error {
		nextTargetPath := filepath.Join(targetPath, strings.TrimPrefix(nextSourcePath, sourcePath))
		if fileInfo == nil {
			return nil
		}

		if !fileInfo.Mode().IsRegular() {
			return nil
		}

		if fileInfo.IsDir() {
			_ = os.MkdirAll(nextTargetPath, os.ModePerm)
			return Copy(nextSourcePath, nextTargetPath, overwrite)
		}

		_, statErr := os.Stat(nextTargetPath)
		if statErr != nil {
			return recursiveCopy.Copy(nextSourcePath, nextTargetPath)
		}

		return nil
	})
}
