package sync

import (
	"os"

	"github.com/devspace-cloud/devspace/sync/remote"
	"github.com/devspace-cloud/devspace/sync/util"
)

// s.fileIndex needs to be locked before this function is called
func shouldRemoveRemote(relativePath string, s *Sync) bool {
	// File / Folder was already deleted from map so event was already processed or should not be processed
	if s.fileIndex.fileMap[relativePath] == nil {
		return false
	}

	// Exclude symbolic links
	if s.fileIndex.fileMap[relativePath].IsSymbolicLink {
		return false
	}

	// Exclude changes on the exclude list
	if s.ignoreMatcher != nil {
		if util.MatchesPath(s.ignoreMatcher, relativePath, s.fileIndex.fileMap[relativePath].IsDirectory) {
			return false
		}
	}

	// Exclude changes on the upload exclude list
	if s.uploadIgnoreMatcher != nil {
		if util.MatchesPath(s.uploadIgnoreMatcher, relativePath, s.fileIndex.fileMap[relativePath].IsDirectory) {
			return false
		}
	}

	return true
}

// s.fileIndex needs to be locked before this function is called
func shouldUpload(s *Sync, fileInformation *FileInformation, isInitial bool) bool {
	// Exclude if stat is nil
	if fileInformation == nil {
		return false
	}

	// Exclude changes on the upload exclude list
	// is not necessary here anymore because it was already
	// checked

	// stat.Mode()&os.ModeSymlink

	// Exclude local symlinks
	if fileInformation.IsSymbolicLink {
		return false
	}

	// Exclude changes on the exclude list
	if s.ignoreMatcher != nil {
		if util.MatchesPath(s.ignoreMatcher, fileInformation.Name, fileInformation.IsDirectory) {
			return false
		}
	}

	// Check if we already tracked the path
	if s.fileIndex.fileMap[fileInformation.Name] != nil {
		// Folder already exists, don't send change
		if fileInformation.IsDirectory {
			return false
		}

		// Exclude symlinks
		if s.fileIndex.fileMap[fileInformation.Name].IsSymbolicLink {
			return false
		}

		if isInitial {
			// File is older locally than remote so don't update remote
			if fileInformation.Mtime <= s.fileIndex.fileMap[fileInformation.Name].Mtime {
				return false
			}
		} else {
			// File did not change or was changed by downstream
			if fileInformation.Mtime == s.fileIndex.fileMap[fileInformation.Name].Mtime && fileInformation.Size == s.fileIndex.fileMap[fileInformation.Name].Size {
				return false
			}
		}
	}

	return true
}

// s.fileIndex needs to be locked before this function is called
func shouldDownload(change *remote.Change, s *Sync) bool {
	// Does file already exist in the filemap?
	if s.fileIndex.fileMap[change.Path] != nil {
		// Don't override folders that exist in the filemap
		if change.IsDir == false {
			// Redownload file if mtime is newer than saved one
			if change.MtimeUnix > s.fileIndex.fileMap[change.Path].Mtime {
				return true
			}

			// Redownload file if size changed && file is not older than the one in the fileMap
			// the mTime check is necessary, because otherwise we would override older local files that
			// are not overridden initially
			if change.MtimeUnix == s.fileIndex.fileMap[change.Path].Mtime && change.Size != s.fileIndex.fileMap[change.Path].Size {
				return true
			}
		}

		return false
	}

	return true
}

// s.fileIndex needs to be locked before this function is called
// A file is only deleted if the following conditions are met:
// - The file name is present in the d.config.fileMap map
// - The file did not change in terms of size and mtime in the d.config.fileMap since we started the collecting changes process
// - The file is present on the filesystem and did not change in terms of size and mtime on the filesystem
func shouldRemoveLocal(absFilepath string, fileInformation *FileInformation, s *Sync) bool {
	if fileInformation == nil {
		s.log.Infof("Skip %s because change is nil", absFilepath)
		return false
	}

	// We don't need to check s.ignoreMatcher, because if a path is ignored it will never be added to the fileMap, because shouldDownload
	// and shouldUpload are always false, and hence it never appears in the fileMap and is not copied to the remove fileMap clone
	// in the beginning of the downstream mainLoop

	// Only delete if mtime and size did not change
	stat, err := os.Stat(absFilepath)
	if err != nil {
		if os.IsNotExist(err) == false {
			s.log.Infof("Skip %s because stat returned %v", absFilepath, err)
		}

		return false
	}

	// We don't delete the file if we haven't tracked it
	if stat != nil && s.fileIndex.fileMap[fileInformation.Name] != nil {
		if stat.IsDir() != s.fileIndex.fileMap[fileInformation.Name].IsDirectory || stat.IsDir() != fileInformation.IsDirectory {
			s.log.Infof("Skip %s because stat returned unequal isdir with fileMap", absFilepath)
			return false
		}

		if fileInformation.IsDirectory == false {
			// We don't delete the file if it has changed in the map since we collected changes
			if fileInformation.Mtime == s.fileIndex.fileMap[fileInformation.Name].Mtime && fileInformation.Size == s.fileIndex.fileMap[fileInformation.Name].Size {
				// We don't delete the file if it has changed on the filesystem meanwhile
				if stat.ModTime().Unix() <= fileInformation.Mtime {
					return true
				}

				s.log.Infof("Skip %s because stat.ModTime() %d is greater than fileInformation.Mtime %d", absFilepath, stat.ModTime().Unix(), fileInformation.Mtime)
			} else {
				s.log.Infof("Skip %s because Mtime (%d and %d) or Size (%d and %d) is unequal between fileInformation and fileMap", absFilepath, fileInformation.Mtime, s.fileIndex.fileMap[fileInformation.Name].Mtime, fileInformation.Size, s.fileIndex.fileMap[fileInformation.Name].Size)
			}
		} else {
			return true
		}
	}

	return false
}
