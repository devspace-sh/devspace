package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/moby/patternmatcher"
	"hash/crc32"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/docker/docker/pkg/longpath"
	"github.com/pkg/errors"
)

// Password hashes the password with sha256 and returns the string
func Password(password string) (string, error) {
	sha256Bytes := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sha256Bytes[:]), nil
}

// File creates the hash value of a file
func File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Directory creates the hash value of a directory
func Directory(path string) (string, error) {
	hash := sha256.New()

	// Stat dir / file
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	// Hash file
	if !fileInfo.IsDir() {
		size := strconv.FormatInt(fileInfo.Size(), 10)
		mTime := strconv.FormatInt(fileInfo.ModTime().UnixNano(), 10)
		_, _ = io.WriteString(hash, path+";"+size+";"+mTime)

		return fmt.Sprintf("%x", hash.Sum(nil)), nil
	}

	// Hash directory
	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// We ignore errors
			return nil
		}

		size := strconv.FormatInt(info.Size(), 10)
		mTime := strconv.FormatInt(info.ModTime().UnixNano(), 10)
		_, _ = io.WriteString(hash, path+";"+size+";"+mTime)

		return nil
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// DirectoryExcludes calculates a hash for a directory and excludes the submitted patterns
func DirectoryExcludes(srcPath string, excludePatterns []string, fast bool) (string, error) {
	srcPath, err := filepath.Abs(srcPath)
	if err != nil {
		return "", err
	}

	hash := sha256.New()

	// Stat dir / file
	fileInfo, err := os.Stat(srcPath)
	if err != nil {
		return "", err
	}

	// Hash file
	if !fileInfo.IsDir() {
		size := strconv.FormatInt(fileInfo.Size(), 10)
		mTime := strconv.FormatInt(fileInfo.ModTime().UnixNano(), 10)
		_, _ = io.WriteString(hash, srcPath+";"+size+";"+mTime)

		return fmt.Sprintf("%x", hash.Sum(nil)), nil
	}

	// Fix the source path to work with long path names. This is a no-op
	// on platforms other than Windows.
	if runtime.GOOS == "windows" {
		srcPath = longpath.AddPrefix(srcPath)
	}

	pm, err := patternmatcher.New(excludePatterns)
	if err != nil {
		return "", err
	}

	// In general we log errors here but ignore them because
	// during e.g. a diff operation the container can continue
	// mutating the filesystem and we can see transient errors
	// from this

	stat, err := os.Lstat(srcPath)
	if err != nil {
		return "", err
	}

	if !stat.IsDir() {
		return "", errors.Errorf("Path %s is not a directory", srcPath)
	}

	include := "."
	seen := make(map[string]bool)

	walkRoot := filepath.Join(srcPath, include)
	err = filepath.Walk(walkRoot, func(filePath string, f os.FileInfo, err error) error {
		if err != nil {
			return errors.Errorf("Hash: Can't stat file %s to hash: %s", srcPath, err)
		}

		relFilePath, err := filepath.Rel(srcPath, filePath)
		if err != nil {
			// Error getting relative path OR we are looking
			// at the source directory path. Skip in both situations.
			return err
		}

		skip := false

		// If "include" is an exact match for the current file
		// then even if there's an "excludePatterns" pattern that
		// matches it, don't skip it. IOW, assume an explicit 'include'
		// is asking for that file no matter what - which is true
		// for some files, like .dockerignore and Dockerfile (sometimes)
		if relFilePath != "." {
			skip, err = pm.MatchesOrParentMatches(relFilePath)
			if err != nil {
				return errors.Errorf("Error matching %s: %v", relFilePath, err)
			}
		}

		if skip {
			// If we want to skip this file and its a directory
			// then we should first check to see if there's an
			// excludes pattern (e.g. !dir/file) that starts with this
			// dir. If so then we can't skip this dir.

			// Its not a dir then so we can just return/skip.
			if !f.IsDir() {
				return nil
			}

			// No exceptions (!...) in patterns so just skip dir
			if !pm.Exclusions() {
				return filepath.SkipDir
			}

			dirSlash := relFilePath + string(filepath.Separator)

			for _, pat := range pm.Patterns() {
				if !pat.Exclusion() {
					continue
				}
				if strings.HasPrefix(pat.String()+string(filepath.Separator), dirSlash) {
					// found a match - so can't skip this dir
					return nil
				}
			}

			// No matching exclusion dir so just skip dir
			return filepath.SkipDir
		}

		if seen[relFilePath] {
			return nil
		}
		seen[relFilePath] = true
		if f.IsDir() {
			// Path is enough
			_, _ = io.WriteString(hash, filePath)
		} else {
			if fast {
				_, _ = io.WriteString(hash, filePath+";"+strconv.FormatInt(f.Size(), 10)+";"+strconv.FormatInt(f.ModTime().Unix(), 10))
			} else {
				// Check file change
				checksum, err := hashFileCRC32(filePath, 0xedb88320)
				if err != nil {
					return nil
				}

				_, _ = io.WriteString(hash, filePath+";"+checksum)
			}
		}

		return nil
	})

	if err != nil {
		return "", errors.Errorf("Error hashing %s: %v", srcPath, err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// StringToNumber hashes a given string to a number
func StringToNumber(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

// String hashes a given string
func String(s string) string {
	hash := sha256.New()
	_, _ = io.WriteString(hash, s)

	return fmt.Sprintf("%x", hash.Sum(nil))
}

func hashFileCRC32(filePath string, polynomial uint32) (string, error) {
	//Initialize an empty return string now in case an error has to be returned
	var returnCRC32String string

	//Open the fhe file located at the given path and check for errors
	file, err := os.Open(filePath)
	if err != nil {
		return returnCRC32String, err
	}

	//Tell the program to close the file when the function returns
	defer file.Close()

	//Create the table with the given polynomial
	tablePolynomial := crc32.MakeTable(polynomial)

	//Open a new hash interface to write the file to
	hash := crc32.New(tablePolynomial)

	//Copy the file in the interface
	if _, err := io.Copy(hash, file); err != nil {
		return returnCRC32String, err
	}

	//Generate the hash
	hashInBytes := hash.Sum(nil)[:]

	//Encode the hash to a string
	returnCRC32String = hex.EncodeToString(hashInBytes)

	//Return the output
	return returnCRC32String, nil
}
