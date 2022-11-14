package sync

import (
	"github.com/loft-sh/devspace/helper/remote"
	"github.com/loft-sh/devspace/helper/server/ignoreparser"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	"github.com/loft-sh/devspace/pkg/util/log"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
)

type initialSyncer struct {
	o *initialSyncOptions
}

type initialSyncOptions struct {
	LocalPath string

	CompareBy latest.InitialSyncCompareBy
	Strategy  latest.InitialSyncStrategy

	IgnoreMatcher         ignoreparser.IgnoreParser
	DownloadIgnoreMatcher ignoreparser.IgnoreParser
	UploadIgnoreMatcher   ignoreparser.IgnoreParser

	UpstreamDisabled   bool
	DownstreamDisabled bool
	FileIndex          *fileIndex

	ApplyRemote func(changes []*FileInformation, remove bool)
	ApplyLocal  func(changes []*remote.Change, force bool) error
	AddSymlink  func(relativePath, absPath string) (os.FileInfo, error)

	UpstreamDone   func()
	DownstreamDone func()

	Log log.Logger
}

func newInitialSyncer(options *initialSyncOptions) *initialSyncer {
	if options.Strategy == "" {
		options.Strategy = latest.InitialSyncStrategyMirrorLocal
	}

	return &initialSyncer{o: options}
}

func (i *initialSyncer) Run(remoteState map[string]*FileInformation, localState map[string]*FileInformation) error {
	// Here we calculate the delta between the remote and local state, the result of this operation
	// are files we should download (new and override) and files we should upload (new and override)
	download := remoteState
	i.o.Log.Debugf("Initial Sync - Calculate Delta from Remote State")
	upload, err := i.CalculateDelta(download, localState)
	i.o.Log.Debugf("Initial Sync - Done Calculating Delta (Download: %d, Upload: %d)", len(download), len(upload))
	if err != nil {
		return errors.Wrap(err, "diff server client")
	}

	// Upstream initial sync
	go func() {
		if !i.o.UpstreamDisabled {
			// Remove remote if mirror local
			if len(download) > 0 && i.o.Strategy == latest.InitialSyncStrategyMirrorLocal {
				deleteRemote := make([]*FileInformation, 0, len(download))
				for _, element := range download {
					if i.o.UploadIgnoreMatcher != nil && i.o.UploadIgnoreMatcher.Matches(element.Name, element.IsDirectory) {
						continue
					}

					deleteRemote = append(deleteRemote, &FileInformation{
						Name:        element.Name,
						IsDirectory: element.IsDirectory,
					})
				}

				i.o.ApplyRemote(deleteRemote, true)
			}

			// Upload remote if not mirror remote
			if len(upload) > 0 {
				if i.o.Strategy == latest.InitialSyncStrategyMirrorRemote {
					// only apply the ones that match the downstream ignore matcher
					changes := []*FileInformation{}
					for _, element := range upload {
						if i.o.DownloadIgnoreMatcher != nil && i.o.DownloadIgnoreMatcher.Matches(element.Name, element.IsDirectory) {
							changes = append(changes, element)
						}
					}
					if len(changes) > 0 {
						i.o.ApplyRemote(changes, false)
					}
				} else {
					i.o.ApplyRemote(upload, false)
				}
			}
		}

		i.o.UpstreamDone()
	}()

	// Download changes if enabled
	if !i.o.DownstreamDisabled {
		// Remove local if mirror remote
		if len(upload) > 0 && i.o.Strategy == latest.InitialSyncStrategyMirrorRemote {
			remoteChanges := make([]*remote.Change, 0, len(upload))
			for _, element := range upload {
				if i.o.DownloadIgnoreMatcher != nil && i.o.DownloadIgnoreMatcher.Matches(element.Name, element.IsDirectory) {
					continue
				}

				remoteChanges = append(remoteChanges, &remote.Change{
					ChangeType:    remote.ChangeType_DELETE,
					Path:          element.Name,
					MtimeUnix:     element.Mtime,
					MtimeUnixNano: element.MtimeNano,
					Mode:          uint32(element.Mode),
					Size:          element.Size,
					IsDir:         element.IsDirectory,
				})
			}

			err = i.o.ApplyLocal(remoteChanges, true)
			if err != nil {
				return errors.Wrap(err, "apply changes")
			}
		}

		// Download local if not mirror local
		if len(download) > 0 {
			if i.o.Strategy == latest.InitialSyncStrategyMirrorLocal {
				// only apply the ones that match the upstream ignore matcher
				remoteChanges := make([]*remote.Change, 0, len(download))
				for _, element := range download {
					if i.o.UploadIgnoreMatcher != nil && i.o.UploadIgnoreMatcher.Matches(element.Name, element.IsDirectory) {
						remoteChanges = append(remoteChanges, &remote.Change{
							ChangeType:    remote.ChangeType_CHANGE,
							Path:          element.Name,
							MtimeUnix:     element.Mtime,
							MtimeUnixNano: element.MtimeNano,
							Mode:          uint32(element.Mode),
							Size:          element.Size,
							IsDir:         element.IsDirectory,
						})
					}
				}
				if len(remoteChanges) > 0 {
					err = i.o.ApplyLocal(remoteChanges, false)
					if err != nil {
						return errors.Wrap(err, "apply changes")
					}
				}
			} else {
				remoteChanges := make([]*remote.Change, 0, len(download))
				for _, element := range download {
					remoteChanges = append(remoteChanges, &remote.Change{
						ChangeType:    remote.ChangeType_CHANGE,
						Path:          element.Name,
						MtimeUnix:     element.Mtime,
						MtimeUnixNano: element.MtimeNano,
						Mode:          uint32(element.Mode),
						Size:          element.Size,
						IsDir:         element.IsDirectory,
					})
				}

				err = i.o.ApplyLocal(remoteChanges, false)
				if err != nil {
					return errors.Wrap(err, "apply changes")
				}
			}
		}
	}

	i.o.DownstreamDone()
	return nil
}

func (i *initialSyncer) CalculateDelta(remoteState map[string]*FileInformation, localState map[string]*FileInformation) ([]*FileInformation, error) {
	strategy := i.o.Strategy
	if i.o.Strategy == latest.InitialSyncStrategyMirrorRemote {
		strategy = latest.InitialSyncStrategyPreferRemote
	} else if i.o.Strategy == latest.InitialSyncStrategyMirrorLocal {
		strategy = latest.InitialSyncStrategyPreferLocal
	}

	return i.deltaState(remoteState, localState, strategy)
}

func (i *initialSyncer) deltaState(remoteState map[string]*FileInformation, localState map[string]*FileInformation, strategy latest.InitialSyncStrategy) ([]*FileInformation, error) {
	changes := make([]*FileInformation, 0, 1024)
	for relativePath, stat := range localState {
		absPath := path.Join(i.o.LocalPath, relativePath)
		ignore := false

		// Exclude changes on the upload exclude list
		if i.o.UploadIgnoreMatcher != nil {
			if i.o.UploadIgnoreMatcher.Matches(relativePath, stat.IsDirectory) {
				i.o.FileIndex.Lock()
				// Add to file map and prevent download if local file is newer than the remote one
				if i.o.FileIndex.fileMap[relativePath] != nil {
					if strategy == latest.InitialSyncStrategyPreferLocal || (strategy == latest.InitialSyncStrategyPreferNewest && i.o.FileIndex.fileMap[relativePath].Mtime < stat.Mtime) {
						// Add it to the fileMap
						i.o.FileIndex.Set(stat)

						delete(remoteState, relativePath)
					}
				}

				i.o.FileIndex.Unlock()
				ignore = true
			}
		}

		// Check for symlinks
		if !ignore && stat.ResolvedLink {
			_, err := i.o.AddSymlink(relativePath, absPath)
			if err != nil {
				return nil, err
			}

			i.o.Log.Infof("Symlink found at %s", absPath)
		}

		// Check if stat is somehow not there
		if i.o.IgnoreMatcher != nil && !i.o.IgnoreMatcher.RequireFullScan() && i.o.IgnoreMatcher.Matches(relativePath, stat.IsDirectory) {
			continue
		}

		if stat.IsDirectory {
			// we don't need to recreate a directory that already exists locally
			delete(remoteState, relativePath)

			// should this directory be added?
			if !ignore && stat.Files == 0 {
				i.o.FileIndex.Lock()
				action := i.decide(stat, strategy)
				i.o.FileIndex.Unlock()

				// This can be only uploadAction or noAction, since this is a directory
				if action == uploadAction {
					changes = append(changes, stat)
				}
			}

			continue
		}

		if !ignore {
			i.o.FileIndex.Lock()
			action := i.decide(stat, strategy)
			i.o.FileIndex.Unlock()
			if action == uploadAction {
				// If we upload the file, don't download it
				delete(remoteState, relativePath)

				// Add file to upload
				// Make sure we use remote mode here
				changes = append(changes, stat)
			} else if action == noAction {
				delete(remoteState, relativePath)
			}
		}
	}

	return changes, nil
}

func (i *initialSyncer) CalculateLocalState(absPath string, localState map[string]*FileInformation, ignore bool) error {
	relativePath := getRelativeFromFullPath(absPath, i.o.LocalPath)

	// We skip files that are suddenly not there anymore
	stat, err := os.Stat(absPath)
	if err != nil {
		return nil
	}

	// Exclude changes on the upload exclude list
	if i.o.UploadIgnoreMatcher != nil {
		if i.o.UploadIgnoreMatcher.Matches(relativePath, stat.IsDir()) {
			ignore = true
		}
	}

	// Check for symlinks
	isSymlink := false
	if !ignore {
		// Retrieve the real stat instead of the symlink one
		lstat, err := os.Lstat(absPath)
		if err == nil && lstat.Mode()&os.ModeSymlink != 0 {
			// Get real path
			targetPath, err := filepath.EvalSymlinks(absPath)
			if err != nil {
				return nil
			}

			stat, err = os.Stat(targetPath)
			if err != nil {
				return nil
			}

			isSymlink = true
		} else if err != nil {
			return nil
		}
	}

	// Check if stat is somehow not there
	if stat == nil {
		return nil
	} else if i.o.IgnoreMatcher != nil && !i.o.IgnoreMatcher.RequireFullScan() && i.o.IgnoreMatcher.Matches(relativePath, stat.IsDir()) {
		return nil
	}

	if stat.IsDir() {
		return i.calculateLocalDirState(absPath, stat, localState, isSymlink, ignore)
	}

	if !ignore {
		// Add file to upload
		localState[relativePath] = &FileInformation{
			Name:           relativePath,
			Mtime:          stat.ModTime().Unix(),
			MtimeNano:      stat.ModTime().UnixNano(),
			Size:           stat.Size(),
			Mode:           stat.Mode(),
			IsDirectory:    false,
			IsSymbolicLink: stat.Mode()&os.ModeSymlink != 0,
			ResolvedLink:   isSymlink,
		}
	}

	return nil
}

func (i *initialSyncer) calculateLocalDirState(absPath string, stat os.FileInfo, localState map[string]*FileInformation, isSymlink, ignore bool) error {
	relativePath := getRelativeFromFullPath(absPath, i.o.LocalPath)
	files, err := os.ReadDir(absPath)
	if err != nil {
		i.o.Log.Infof("Couldn't read dir %s: %v", absPath, err)
		return nil
	}

	if relativePath != "" && !ignore && stat != nil {
		localState[relativePath] = &FileInformation{
			Name:           relativePath,
			Mtime:          stat.ModTime().Unix(),
			MtimeNano:      stat.ModTime().UnixNano(),
			Size:           stat.Size(),
			Mode:           stat.Mode(),
			IsDirectory:    true,
			IsSymbolicLink: stat.Mode()&os.ModeSymlink != 0,
			ResolvedLink:   isSymlink,
			Files:          len(files),
		}
	}

	for _, dirEntry := range files {
		f, err := dirEntry.Info()
		if err != nil {
			continue
		}

		if fsutil.IsRecursiveSymlink(f, filepath.Join(absPath, f.Name())) {
			i.o.Log.Debugf("Found recursive symlink at %v", filepath.Join(absPath, f.Name()))
			continue
		}

		err = i.CalculateLocalState(filepath.Join(absPath, f.Name()), localState, ignore)
		if err != nil {
			return errors.Wrap(err, f.Name())
		}
	}

	return nil
}

type action int

const (
	uploadAction   action = iota
	downloadAction action = iota
	noAction       action = iota
)

func (i *initialSyncer) decide(fileInformation *FileInformation, strategy latest.InitialSyncStrategy) action {
	// Exclude if stat is nil
	if fileInformation == nil {
		return downloadAction
	}

	// Exclude local symlinks
	if fileInformation.IsSymbolicLink {
		return noAction
	}

	// Exclude changes on the exclude list
	if i.o.IgnoreMatcher != nil {
		if i.o.IgnoreMatcher.Matches(fileInformation.Name, fileInformation.IsDirectory) {
			return noAction
		}
	}

	// Check if we already tracked the path
	if i.o.FileIndex.fileMap[fileInformation.Name] != nil {
		// Folder already exists, don't send change
		if fileInformation.IsDirectory {
			return noAction
		}

		// Exclude symlinks
		if i.o.FileIndex.fileMap[fileInformation.Name].IsSymbolicLink {
			return noAction
		}

		// File did not change or was changed by downstream
		if fileInformation.Size == i.o.FileIndex.fileMap[fileInformation.Name].Size {
			if strategy == latest.InitialSyncStrategyPreferLocal && !equalFilePermissions(fileInformation.Mode, i.o.FileIndex.fileMap[fileInformation.Name].Mode) {
				return uploadAction
			}

			if fileInformation.Mtime == i.o.FileIndex.fileMap[fileInformation.Name].Mtime {
				return noAction
			} else if i.o.CompareBy == latest.InitialSyncCompareBySize {
				return noAction
			}
		}

		// Okay we have a conflict so now we decide based on the given strategy
		switch strategy {
		case latest.InitialSyncStrategyPreferLocal:
			return uploadAction
		case latest.InitialSyncStrategyPreferRemote:
			return downloadAction
		case latest.InitialSyncStrategyPreferNewest:
			if fileInformation.Mtime == i.o.FileIndex.fileMap[fileInformation.Name].Mtime {
				return noAction
			} else if fileInformation.Mtime > i.o.FileIndex.fileMap[fileInformation.Name].Mtime {
				return uploadAction
			} else {
				return downloadAction
			}
		case latest.InitialSyncStrategyKeepAll:
			return noAction
		}
	}

	return uploadAction
}
