package hash

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/pkg/fileutils"
	"github.com/docker/docker/pkg/longpath"
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

// DirectoryExcludes calculates a hash for a directory and excludes the submitted patterns
func DirectoryExcludes(srcPath string, excludePatterns []string) (string, error) {
	hash := sha256.New()

	// Fix the source path to work with long path names. This is a no-op
	// on platforms other than Windows.
	srcPath = longpath.AddPrefix(srcPath)

	pm, err := fileutils.NewPatternMatcher(excludePatterns)
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
		return "", fmt.Errorf("Path %s is not a directory", srcPath)
	}

	include := "."
	seen := make(map[string]bool)

	walkRoot := filepath.Join(srcPath, include)
	err = filepath.Walk(walkRoot, func(filePath string, f os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("Hash: Can't stat file %s to hash: %s", srcPath, err)
		}

		relFilePath, err := filepath.Rel(srcPath, filePath)
		if err != nil {
			// Error getting relative path OR we are looking
			// at the source directory path. Skip in both situations.
			return err
		}

		if include == "." && relFilePath != "." {
			relFilePath = strings.Join([]string{".", relFilePath}, string(filepath.Separator))
		}

		skip := false

		// If "include" is an exact match for the current file
		// then even if there's an "excludePatterns" pattern that
		// matches it, don't skip it. IOW, assume an explicit 'include'
		// is asking for that file no matter what - which is true
		// for some files, like .dockerignore and Dockerfile (sometimes)
		if relFilePath != "." {
			skip, err = pm.Matches(relFilePath)
			if err != nil {
				return fmt.Errorf("Error matching %s: %v", relFilePath, err)
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

		size := strconv.FormatInt(f.Size(), 10)
		mTime := strconv.FormatInt(f.ModTime().UnixNano(), 10)
		io.WriteString(hash, filePath+";"+size+";"+mTime)

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("Error hashing %s: %v", srcPath, err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
