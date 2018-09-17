package fsutil

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	recursiveCopy "github.com/otiai10/copy"
)

//WriteToFile writes data to a file
func WriteToFile(data []byte, filePath string) error {
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filePath, data, 0666)
}

//Copy copies a file to a destination path
func Copy(sourcePath string, targetPath string, overwrite bool) error {
	if overwrite {
		return recursiveCopy.Copy(sourcePath, targetPath)
	}
	pathSeparator := string(os.PathSeparator)

	if pathSeparator == "/" {
		sourcePath = strings.Replace(sourcePath, "\\", pathSeparator, -1)
	} else {
		sourcePath = strings.Replace(sourcePath, "/", pathSeparator, -1)
	}

	return filepath.Walk(sourcePath, func(nextSourcePath string, fileInfo os.FileInfo, err error) error {
		nextTargetPath := filepath.Join(targetPath, strings.TrimPrefix(nextSourcePath, sourcePath))

		if !fileInfo.Mode().IsRegular() {
			return nil
		}

		if fileInfo.IsDir() {
			os.MkdirAll(nextTargetPath, os.ModePerm)

			return Copy(nextSourcePath, nextTargetPath, overwrite)
		}
		_, statErr := os.Stat(nextTargetPath)

		if statErr != nil {
			return recursiveCopy.Copy(nextSourcePath, nextTargetPath)
		}
		return nil
	})
}

//ReadFile reads a file
func ReadFile(path string, limit int64) ([]byte, error) {
	if limit <= 0 {
		return ioutil.ReadFile(path)
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

//GetHomeDir returns the home variable
func GetHomeDir() string {
	homedir := os.Getenv("HOME")

	if homedir != "" {
		return homedir
	}
	return os.Getenv("USERPROFILE")
}

//GetCurrentGofileDir returns the parent dir of the file with the source code that called this method
func GetCurrentGofileDir() string {
	_, filename, _, _ := runtime.Caller(1)

	return path.Dir(filename)
}

//GetCurrentGofile returns the file with the source code that called this method
func GetCurrentGofile() string {
	_, filename, _, _ := runtime.Caller(1)

	return filename
}
