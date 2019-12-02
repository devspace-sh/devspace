package sync

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/juju/ratelimit"
	"github.com/pkg/errors"
	gitignore "github.com/sabhiram/go-gitignore"

	"github.com/devspace-cloud/devspace/sync/remote"
	"github.com/devspace-cloud/devspace/sync/util"
	"github.com/rjeczalik/notify"
)

type upstream struct {
	events    chan notify.EventInfo
	symlinks  map[string]*Symlink
	interrupt chan bool
	sync      *Sync

	reader io.ReadCloser
	writer io.WriteCloser
	client remote.UpstreamClient
}

const removeFilesBufferSize = 64

// newUpstream creates a new upstream handler with the given parameters
func newUpstream(reader io.ReadCloser, writer io.WriteCloser, sync *Sync) (*upstream, error) {
	var (
		clientReader io.Reader = reader
		clientWriter io.Writer = writer
	)

	// Apply limits if specified
	if sync.Options.DownstreamLimit > 0 {
		clientReader = ratelimit.Reader(reader, ratelimit.NewBucketWithRate(float64(sync.Options.DownstreamLimit), sync.Options.DownstreamLimit))
	}
	if sync.Options.UpstreamLimit > 0 {
		clientWriter = ratelimit.Writer(writer, ratelimit.NewBucketWithRate(float64(sync.Options.UpstreamLimit), sync.Options.UpstreamLimit))
	}

	// Create client
	conn, err := util.NewClientConnection(clientReader, clientWriter)
	if err != nil {
		return nil, errors.Wrap(err, "new client connection")
	}

	return &upstream{
		events:    make(chan notify.EventInfo, 3000), // High buffer size so we don't miss any fsevents if there are a lot of changes
		symlinks:  make(map[string]*Symlink),
		interrupt: make(chan bool, 1),
		sync:      sync,

		reader: reader,
		writer: writer,
		client: remote.NewUpstreamClient(conn),
	}, nil
}

func (u *upstream) mainLoop() error {
	for {
		var (
			changes      []*FileInformation
			changeAmount = 0
		)

		for {
			select {
			case <-u.interrupt:
				return nil
			case event, ok := <-u.events:
				if ok == false {
					return nil
				}

				events := make([]notify.EventInfo, 0, 10)
				events = append(events, event)

				// We need this loop to catch up if we got a lot of change events
				for eventsLeft := true; eventsLeft == true; {
					select {
					case event := <-u.events:
						events = append(events, event)
						break
					default:
						eventsLeft = false
						break
					}
				}

				fileInformations, err := u.getfileInformationFromEvent(events)
				if err != nil {
					return errors.Wrap(err, "get file information from event")
				}

				changes = append(changes, fileInformations...)
			case <-time.After(time.Millisecond * 600):
				break
			}

			// We gather changes till there are no more changes for 1 second
			if changeAmount == len(changes) && changeAmount > 0 {
				break
			}

			changeAmount = len(changes)
		}

		err := u.applyChanges(changes)
		if err != nil {
			return errors.Wrap(err, "apply changes")
		}
	}
}

func (u *upstream) getfileInformationFromEvent(events []notify.EventInfo) ([]*FileInformation, error) {
	u.sync.fileIndex.fileMapMutex.Lock()
	defer u.sync.fileIndex.fileMapMutex.Unlock()

	changes := make([]*FileInformation, 0, len(events))
	for _, event := range events {
		fileInfo, ok := event.(*FileInformation)

		if ok {
			changes = append(changes, fileInfo)
		} else {
			fullpath := event.Path()
			relativePath := getRelativeFromFullPath(fullpath, u.sync.LocalPath)

			// Determine what kind of change we got (Create or Remove)
			newChange, err := u.evaluateChange(relativePath, fullpath)
			if err != nil {
				return nil, errors.Wrap(err, "evaluate change")
			}

			if newChange != nil {
				changes = append(changes, newChange)
			}
		}
	}

	return changes, nil
}

func (u *upstream) evaluateChange(relativePath, fullpath string) (*FileInformation, error) {
	stat, err := os.Stat(fullpath)

	// File / Folder exist -> Create File or Folder
	// if File / Folder does not exist, we create a new remove change
	if err == nil {
		// Exclude changes on the upload exclude list
		if u.sync.uploadIgnoreMatcher != nil {
			if util.MatchesPath(u.sync.uploadIgnoreMatcher, relativePath, stat.IsDir()) {
				// Add to file map and prevent download if local file is newer than the remote one
				if u.sync.fileIndex.fileMap[relativePath] != nil && u.sync.fileIndex.fileMap[relativePath].Mtime < stat.ModTime().Unix() {
					// Add it to the fileMap
					u.sync.fileIndex.fileMap[relativePath] = &FileInformation{
						Name:        relativePath,
						Mtime:       stat.ModTime().Unix(),
						Size:        stat.Size(),
						IsDirectory: stat.IsDir(),
					}
				}

				return nil, nil
			}
		}

		// Check if symbolic link
		lstat, err := os.Lstat(fullpath)
		if err == nil && lstat.Mode()&os.ModeSymlink != 0 {
			_, symlinkExists := u.sync.upstream.symlinks[fullpath]

			// Add symlink to map
			stat, err = u.sync.upstream.AddSymlink(relativePath, fullpath)
			if err != nil {
				return nil, errors.Wrap(err, "add symlink")
			}
			if stat == nil {
				return nil, nil
			}

			// Only crawl if symlink wasn't there before and it is a directory
			if symlinkExists == false && stat.IsDir() {
				// Crawl all linked files & folders
				err = u.symlinks[fullpath].Crawl()
				if err != nil {
					return nil, errors.Wrap(err, "crawl symlink")
				}
			}
		} else if err != nil {
			return nil, nil
		} else if stat == nil {
			return nil, nil
		}

		fileInfo := &FileInformation{
			Name:           relativePath,
			Mtime:          stat.ModTime().Unix(),
			MtimeNano:      stat.ModTime().UnixNano(),
			Size:           stat.Size(),
			IsDirectory:    stat.IsDir(),
			IsSymbolicLink: stat.Mode()&os.ModeSymlink != 0,
		}
		if shouldUpload(u.sync, fileInfo, false) {
			// New Create Task
			return fileInfo, nil
		}
	} else {
		// Remove symlinks
		u.RemoveSymlinks(fullpath)

		// Check if we should remove path remote
		if shouldRemoveRemote(relativePath, u.sync) {
			// New Remove Task
			return &FileInformation{
				Name: relativePath,
			}, nil
		}
	}

	return nil, nil
}

func (u *upstream) AddSymlink(relativePath, absPath string) (os.FileInfo, error) {
	// Get real path
	targetPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		u.sync.log.Infof("Warning: resolving symlink of %s: %v", absPath, err)
		return nil, nil // errors.Errorf("Error resolving symlink of %s: %v", absPath, err)
	}

	stat, err := os.Stat(targetPath)
	if err != nil {
		u.sync.log.Infof("Warning: stating symlink %s: %v", targetPath, err)
		return nil, nil // errors.Errorf("Error stating symlink %s: %v", targetPath, err)
	}

	// Check if we already added the symlink
	if _, ok := u.symlinks[absPath]; ok {
		return stat, nil
	}

	// Check if symlink is ignored
	if u.sync.ignoreMatcher != nil {
		if util.MatchesPath(u.sync.ignoreMatcher, relativePath, stat.IsDir()) {
			return nil, nil
		}
	}

	symlink, err := NewSymlink(u, absPath, targetPath, stat.IsDir())
	if err != nil {
		return nil, errors.Errorf("Cannot create symlink object for %s: %v", absPath, err)
	}

	u.symlinks[absPath] = symlink

	return stat, nil
}

func (u *upstream) RemoveSymlinks(absPath string) {
	for key, symlink := range u.symlinks {
		if key == absPath || strings.Index(filepath.ToSlash(key)+"/", filepath.ToSlash(absPath)) == 0 {
			symlink.Stop()
			delete(u.symlinks, key)
		}
	}
}

func (u *upstream) applyChanges(changes []*FileInformation) error {
	var creates []*FileInformation
	var removes []*FileInformation

	// First we cluster changes into remove and create changes
	for _, element := range changes {
		// We determine if a change is a remove or create change by setting
		// the mtime to 0 in the fileinformation for remove changes
		if element.Mtime > 0 {
			creates = append(creates, element)
		} else {
			removes = append(removes, element)
		}
	}

	// Apply removes
	if len(removes) > 0 {
		err := u.applyRemoves(removes)
		if err != nil {
			return errors.Wrap(err, "apply removes")
		}
	}

	// Apply creates
	if len(creates) > 0 {
		for i := 0; i < syncRetries; i++ {
			err := u.applyCreates(creates)
			if err == nil {
				break
			} else if i+1 >= syncRetries {
				return errors.Wrap(err, "apply creates")
			}

			u.sync.log.Infof("Upstream - Retry upload because of error: %v", errors.Cause(err))

			creates = u.updateUploadChanges(creates)
			if len(creates) == 0 {
				break
			}
		}
	}

	u.sync.log.Infof("Upstream - Successfully processed %d change(s)", len(changes))
	return nil
}

func (u *upstream) updateUploadChanges(files []*FileInformation) []*FileInformation {
	u.sync.fileIndex.fileMapMutex.Lock()
	defer u.sync.fileIndex.fileMapMutex.Unlock()

	newChanges := make([]*FileInformation, 0, len(files))
	for _, change := range files {
		if shouldUpload(u.sync, change, false) {
			newChanges = append(newChanges, change)
		}
	}

	return newChanges
}

func (u *upstream) applyCreates(files []*FileInformation) error {
	size := int64(0)
	for _, c := range files {
		if c.IsDirectory {
			// Print changes
			if u.sync.Options.Verbose || len(files) <= 3 {
				u.sync.log.Infof("Upstream - Upload Folder %s", c.Name)
			}
		} else {
			if u.sync.Options.Verbose || len(files) <= 3 {
				u.sync.log.Infof("Upstream - Upload File %s", c.Name)
			}

			size += c.Size
		}
	}

	u.sync.log.Infof("Upstream - Upload %d create changes (size %d)", len(files), size)

	// Create combined exclude paths
	excludePaths := make([]string, 0, len(u.sync.Options.ExcludePaths)+len(u.sync.Options.UploadExcludePaths))
	excludePaths = append(excludePaths, u.sync.Options.ExcludePaths...)
	excludePaths = append(excludePaths, u.sync.Options.UploadExcludePaths...)

	ignoreMatcher, err := CompilePaths(excludePaths)
	if err != nil {
		return errors.Wrap(err, "compile paths")
	}

	// Create a pipe for reading and writing
	reader, writer, err := os.Pipe()
	if err != nil {
		return errors.Wrap(err, "create pipe")
	}

	defer reader.Close()
	defer writer.Close()

	// Upload files
	u.sync.fileIndex.fileMapMutex.Lock()
	defer u.sync.fileIndex.fileMapMutex.Unlock()

	errorChan := make(chan error)
	go func() {
		errorChan <- u.compress(writer, files, ignoreMatcher)
	}()

	err = u.uploadArchive(reader)
	if err != nil {
		return errors.Wrap(err, "upload archive")
	}

	return <-errorChan
}

func (u *upstream) compress(writer io.WriteCloser, files []*FileInformation, ignoreMatcher gitignore.IgnoreParser) error {
	defer writer.Close()

	// Use compression
	gw := gzip.NewWriter(writer)
	defer gw.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	writtenFiles := make(map[string]*FileInformation)
	for _, file := range files {
		if writtenFiles[file.Name] == nil {
			err := RecursiveTar(u.sync.LocalPath, file.Name, writtenFiles, tarWriter, ignoreMatcher)
			if err != nil {
				return errors.Wrap(err, "recursive tar")
			}
		}
	}

	// Update sync filemap
	for _, element := range writtenFiles {
		u.sync.fileIndex.CreateDirInFileMap(path.Dir(element.Name))
		u.sync.fileIndex.fileMap[element.Name] = element
	}

	return nil
}

func (u *upstream) uploadArchive(reader io.Reader) error {
	// Create upload client
	uploadClient, err := u.client.Upload(context.Background())
	if err != nil {
		return errors.Wrap(err, "upload")
	}

	buf := make([]byte, 16*1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			err := uploadClient.Send(&remote.Chunk{
				Content: buf[:n],
			})
			if err != nil {
				return errors.Wrap(err, "upload send")
			}
		}

		if err == io.EOF {
			_, err := uploadClient.CloseAndRecv()
			if err != nil {
				return errors.Wrap(err, "after upload")
			}

			break
		} else if err != nil {
			return errors.Wrap(err, "read tar")
		}
	}

	return nil
}

func (u *upstream) applyRemoves(files []*FileInformation) error {
	u.sync.fileIndex.fileMapMutex.Lock()
	defer u.sync.fileIndex.fileMapMutex.Unlock()

	u.sync.log.Infof("Upstream - Handling %d removes", len(files))
	fileMap := u.sync.fileIndex.fileMap

	removeClient, err := u.client.Remove(context.Background())
	if err != nil {
		return errors.Wrap(err, "remove client")
	}

	sendFiles := make([]string, 0, removeFilesBufferSize)
	for _, file := range files {
		sendFiles = append(sendFiles, file.Name)

		if fileMap[file.Name] != nil {
			if fileMap[file.Name].IsDirectory {
				u.sync.fileIndex.RemoveDirInFileMap(file.Name)
			} else {
				delete(fileMap, file.Name)
			}

			// Print changes
			if u.sync.Options.Verbose || len(files) <= 3 {
				u.sync.log.Infof("Upstream - Remove %s", file.Name)
			}
		}

		if len(sendFiles) >= removeFilesBufferSize {
			err = removeClient.Send(&remote.Paths{
				Paths: sendFiles,
			})
			if err != nil {
				return errors.Wrap(err, "send paths")
			}

			sendFiles = make([]string, 0, removeFilesBufferSize)
		}
	}

	if len(sendFiles) > 0 {
		err = removeClient.Send(&remote.Paths{
			Paths: sendFiles,
		})
		if err != nil {
			return errors.Wrap(err, "send paths")
		}
	}

	_, err = removeClient.CloseAndRecv()
	if err != nil {
		return errors.Wrap(err, "after deletes")
	}

	return nil
}
