package sync

import (
	"os"
)

// s.fileIndex needs to be locked before this function is called
func shouldRemoveRemote(relativePath string, s *SyncConfig) bool {
	// Exclude changes on the exclude list
	if s.ignoreMatcher != nil {
		if s.ignoreMatcher.MatchesPath(relativePath) {
			return false
		}
	}

	// Exclude changes on the upload exclude list
	if s.uploadIgnoreMatcher != nil {
		if s.uploadIgnoreMatcher.MatchesPath(relativePath) {
			return false
		}
	}

	// File / Folder was already deleted from map so event was already processed or should not be processed
	if s.fileIndex.fileMap[relativePath] == nil {
		return false
	}

	// Exclude symbolic links
	if s.fileIndex.fileMap[relativePath].IsSymbolicLink {
		return false
	}

	return true
}

// s.fileIndex needs to be locked before this function is called
func shouldUpload(relativePath string, stat os.FileInfo, s *SyncConfig, isInitial bool) bool {
	// Exclude if stat is nil
	if stat == nil {
		return false
	}

	// Exclude changes on the exclude list
	if s.ignoreMatcher != nil {
		if s.ignoreMatcher.MatchesPath(relativePath) {
			return false
		}
	}

	// Exclude changes on the upload exclude list
	if s.uploadIgnoreMatcher != nil {
		if s.uploadIgnoreMatcher.MatchesPath(relativePath) {
			// Add to file map and prevent download if local file is newer than the remote one
			if s.fileIndex.fileMap[relativePath] != nil && s.fileIndex.fileMap[relativePath].Mtime < ceilMtime(stat.ModTime()) {
				// Add it to the fileMap
				s.fileIndex.fileMap[relativePath] = &fileInformation{
					Name:        relativePath,
					Mtime:       ceilMtime(stat.ModTime()),
					Size:        stat.Size(),
					IsDirectory: stat.IsDir(),
				}
			}

			return false
		}
	}

	// Exclude local symlinks
	if stat.Mode()&os.ModeSymlink != 0 {
		return false
	}

	// Check if we already tracked the path
	if s.fileIndex.fileMap[relativePath] != nil {
		// Folder already exists
		if stat.IsDir() {
			// We want to initially walk over all files therefore we return true for a directory
			// Later on a created directory locally that already exists in the fileMap should be ignored
			return isInitial
		}

		// Exclude symlinks
		if s.fileIndex.fileMap[relativePath].IsSymbolicLink {
			return false
		}

		if isInitial {
			// File is older locally than remote so don't update remote
			if ceilMtime(stat.ModTime()) <= s.fileIndex.fileMap[relativePath].Mtime+1 {
				return false
			}
		} else {
			// File did not change or was changed by downstream
			if ceilMtime(stat.ModTime()) == s.fileIndex.fileMap[relativePath].Mtime && stat.Size() == s.fileIndex.fileMap[relativePath].Size {
				return false
			}
		}
	}

	return true
}

// s.fileIndex needs to be locked before this function is called
func shouldDownload(fileInformation *fileInformation, s *SyncConfig) bool {
	// Exclude files on the exclude list
	if s.ignoreMatcher != nil {
		if s.ignoreMatcher.MatchesPath(fileInformation.Name) {
			return false
		}
	}

	// Update mode, gid & uid if exists
	if s.fileIndex.fileMap[fileInformation.Name] != nil {
		s.fileIndex.fileMap[fileInformation.Name].RemoteMode = fileInformation.RemoteMode
		s.fileIndex.fileMap[fileInformation.Name].RemoteGID = fileInformation.RemoteGID
		s.fileIndex.fileMap[fileInformation.Name].RemoteUID = fileInformation.RemoteUID
	}

	// Exclude files on the exclude list
	if s.downloadIgnoreMatcher != nil {
		if s.downloadIgnoreMatcher.MatchesPath(fileInformation.Name) {
			return false
		}
	}

	// Exclude symlinks
	if fileInformation.IsSymbolicLink {
		// Add them to the fileMap though
		s.fileIndex.fileMap[fileInformation.Name] = fileInformation
		return false
	}

	// Does file already exist in the filemap?
	if s.fileIndex.fileMap[fileInformation.Name] != nil {
		// Don't override folders that exist in the filemap
		if fileInformation.IsDirectory == false {
			// Redownload file if mtime is newer than saved one
			if fileInformation.Mtime > s.fileIndex.fileMap[fileInformation.Name].Mtime {
				return true
			}

			// Redownload file if size changed && file is not older than the one in the fileMap
			// the mTime check is necessary, because otherwise we would override older local files that
			// are not overridden initially
			if fileInformation.Mtime == s.fileIndex.fileMap[fileInformation.Name].Mtime && fileInformation.Size != s.fileIndex.fileMap[fileInformation.Name].Size {
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
func shouldRemoveLocal(absFilepath string, fileInformation *fileInformation, s *SyncConfig) bool {
	if fileInformation == nil {
		return false
	}

	// Exclude files on the exclude list
	if s.downloadIgnoreMatcher != nil {
		if s.downloadIgnoreMatcher.MatchesPath(fileInformation.Name) {
			return false
		}
	}

	// Only delete if mtime and size did not change
	stat, err := os.Stat(absFilepath)
	if err != nil {
		return false
	}

	// We don't delete the file if we haven't tracked it
	if stat != nil && s.fileIndex.fileMap[fileInformation.Name] != nil {
		if stat.IsDir() != s.fileIndex.fileMap[fileInformation.Name].IsDirectory || stat.IsDir() != fileInformation.IsDirectory {
			return false
		}

		if fileInformation.IsDirectory == false {
			// We don't delete the file if it has changed in the map since we collected changes
			if fileInformation.Mtime == s.fileIndex.fileMap[fileInformation.Name].Mtime && fileInformation.Size == s.fileIndex.fileMap[fileInformation.Name].Size {
				// We don't delete the file if it has changed on the filesystem meanwhile
				if ceilMtime(stat.ModTime()) <= fileInformation.Mtime {
					return true
				}
			}
		} else {
			return true
		}
	}

	return false
}
