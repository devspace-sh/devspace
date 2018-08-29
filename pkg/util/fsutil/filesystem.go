package fsutil

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"

	recursiveCopy "github.com/otiai10/copy"
)

//WriteToFile writes data to a file
func WriteToFile(data []byte, filePath string) error {
	os.MkdirAll(filepath.Dir(filePath), os.ModePerm)

	file, fopenErr := os.OpenFile(filePath, os.O_CREATE, os.ModePerm)

	defer file.Close()

	if fopenErr != nil {
		return fopenErr
	}
	fileWriter := bufio.NewWriter(file)
	_, fwriteErr := fileWriter.Write(data)

	if fwriteErr != nil {
		return fwriteErr
	}
	flushErr := fileWriter.Flush()

	if flushErr != nil {
		return flushErr
	}
	return nil
}

//Copy copies a file to a destination path
func Copy(sourcePath string, targetPath string) error {
	return recursiveCopy.Copy(sourcePath, targetPath)
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
