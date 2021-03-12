package sync

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"github.com/loft-sh/devspace/helper/server/ignoreparser"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/juju/ratelimit"
	"github.com/loft-sh/devspace/helper/remote"
	"github.com/loft-sh/devspace/helper/util"
	"github.com/pkg/errors"
	"github.com/syncthing/notify"
)

type upstream struct {
	events    chan notify.EventInfo
	symlinks  map[string]*Symlink
	interrupt chan bool
	sync      *Sync

	reader io.ReadCloser
	writer io.WriteCloser
	client remote.UpstreamClient

	isBusy      bool
	isBusyMutex sync.Mutex

	eventBuffer      []notify.EventInfo
	eventBufferMutex sync.Mutex

	workingDirectory string

	ignoreMatcher ignoreparser.IgnoreParser
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

	// Create combined exclude paths
	excludePaths := make([]string, 0, len(sync.Options.ExcludePaths)+len(sync.Options.UploadExcludePaths))
	excludePaths = append(excludePaths, sync.Options.ExcludePaths...)
	excludePaths = append(excludePaths, sync.Options.UploadExcludePaths...)

	ignoreMatcher, err := ignoreparser.CompilePaths(excludePaths)
	if err != nil {
		return nil, errors.Wrap(err, "compile paths")
	}

	workingDirectory, _ := os.Getwd()
	return &upstream{
		events:      make(chan notify.EventInfo, 1000), // High buffer size so we don't miss any fsevents if there are a lot of changes
		eventBuffer: make([]notify.EventInfo, 0, 64),
		symlinks:    make(map[string]*Symlink),
		interrupt:   make(chan bool, 1),
		sync:        sync,
		isBusy:      true,

		reader: reader,
		writer: writer,
		client: remote.NewUpstreamClient(conn),

		workingDirectory: workingDirectory,
		ignoreMatcher:    ignoreMatcher,
	}, nil
}

func (u *upstream) IsBusy() bool {
	u.isBusyMutex.Lock()
	defer u.isBusyMutex.Unlock()

	return u.isBusy
}

func (u *upstream) startEventsLoop(doneChan chan struct{}) {
	go func() {
		for {
			select {
			case <-doneChan:
				return
			case event, ok := <-u.events:
				if ok == false {
					return
				}

				// We need this loop to catch up if we got a lot of change events
				u.eventBufferMutex.Lock()
				u.eventBuffer = append(u.eventBuffer, event)
				for eventsLeft := true; eventsLeft == true; {
					select {
					case event := <-u.events:
						u.eventBuffer = append(u.eventBuffer, event)
						break
					default:
						eventsLeft = false
						break
					}
				}
				u.eventBufferMutex.Unlock()
			}
		}
	}()
}

func (u *upstream) getEvents() []notify.EventInfo {
	var eventsRef []notify.EventInfo
	u.eventBufferMutex.Lock()
	defer u.eventBufferMutex.Unlock()

	// exchange buffer if we got events
	if len(u.eventBuffer) > 0 {
		eventsRef = u.eventBuffer
		u.eventBuffer = make([]notify.EventInfo, 0, 64)
	}

	return eventsRef
}

func (u *upstream) mainLoop() error {
	doneChan := make(chan struct{})
	defer close(doneChan)

	// start collecting events
	u.startEventsLoop(doneChan)

	for {
		var (
			changes      []*FileInformation
			changeAmount = 0
		)

		// gather changes
		for {
			select {
			case <-u.interrupt:
				return nil
			case <-time.After(time.Millisecond * 600):
				break
			}

			// retrieve the newest events
			events := u.getEvents()
			if len(events) > 0 {
				fileInformations, err := u.getfileInformationFromEvent(events)
				if err != nil {
					return errors.Wrap(err, "get file information from event")
				}

				changes = append(changes, fileInformations...)
			}

			// We gather changes till there are no more changes or
			// a certain amount of changes is reached
			if len(changes) > 50000 || (changeAmount == len(changes) && changeAmount > 0) {
				break
			}

			changeAmount = len(changes)
			if changeAmount == 0 && len(u.events) == 0 {
				u.isBusyMutex.Lock()
				if len(u.events) == 0 {
					u.isBusy = false
				}
				u.isBusyMutex.Unlock()
			}
		}

		// apply the changes
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

		// if the change is sent from the initial sync don't evaluate it
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
		if shouldUpload(u.sync, fileInfo) {
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

			u.sync.log.Infof("Upstream - Retry upload because of error: %v", err)
			creates = u.updateUploadChanges(creates)
			if len(creates) == 0 {
				break
			}
		}
	}

	u.sync.log.Infof("Upstream - Successfully processed %d change(s)", len(changes))

	// Restart container if needed
	return u.RestartContainer()
}

func (u *upstream) RestartContainer() error {
	if u.sync.Options.RestartContainer {
		u.sync.log.Info("Upstream - Restarting container")

		_, err := u.client.RestartContainer(context.Background(), &remote.Empty{})
		if err != nil {
			return errors.Wrap(err, "restart container")
		}
	}

	return nil
}

func (u *upstream) updateUploadChanges(files []*FileInformation) []*FileInformation {
	u.sync.fileIndex.fileMapMutex.Lock()
	defer u.sync.fileIndex.fileMapMutex.Unlock()

	newChanges := make([]*FileInformation, 0, len(files))
	for _, change := range files {
		if shouldUpload(u.sync, change) {
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
				u.sync.log.Infof("Upstream - Upload Folder '%s'", u.getRelativeUpstreamPath(c.Name))
			}
		} else {
			if u.sync.Options.Verbose || len(files) <= 3 {
				u.sync.log.Infof("Upstream - Upload File '%s'", u.getRelativeUpstreamPath(c.Name))
			}

			size += c.Size
		}
	}
	u.sync.log.Infof("Upstream - Upload %d create change(s) (Uncompressed ~%0.2f KB)", len(files), float64(size)/1024.0)

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

	var archiver *Archiver
	errorChan := make(chan error)
	go func() {
		var compressErr error
		archiver, compressErr = u.compress(writer, files, u.ignoreMatcher)
		errorChan <- compressErr
	}()

	// upload the archive
	err = u.uploadArchive(reader)
	if err != nil {
		return errors.Wrap(err, "upload archive")
	}

	// check if there was a compressing error
	err = <-errorChan
	if err != nil {
		return errors.Wrap(err, "compress archive")
	}

	// finally update written files
	for _, element := range archiver.WrittenFiles() {
		u.sync.fileIndex.CreateDirInFileMap(path.Dir(element.Name))
		u.sync.fileIndex.fileMap[element.Name] = element
	}

	return nil
}

func (u *upstream) compress(writer io.WriteCloser, files []*FileInformation, ignoreMatcher ignoreparser.IgnoreParser) (*Archiver, error) {
	defer writer.Close()

	// Use compression
	gw := gzip.NewWriter(writer)
	defer gw.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	// Archive the given files
	archiver := NewArchiver(u.sync.LocalPath, tarWriter, ignoreMatcher)
	for _, file := range files {
		err := archiver.AddToArchive(file.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "compress %s", file.Name)
		}
	}

	return archiver, nil
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
				_, recvErr := uploadClient.CloseAndRecv()
				if recvErr != nil {
					return errors.Wrap(recvErr, "upload send")
				}

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
				u.sync.log.Infof("Upstream - Remove '%s'", u.getRelativeUpstreamPath(file.Name))
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

func (u *upstream) getRelativeUpstreamPath(uploadPath string) string {
	if uploadPath == "" {
		return uploadPath
	}

	uploadPath = uploadPath[1:]
	out, err := filepath.Rel(filepath.FromSlash(u.workingDirectory), filepath.Join(filepath.FromSlash(u.sync.LocalPath), filepath.FromSlash(uploadPath)))
	if err != nil {
		return "./" + uploadPath
	}

	return out
}
