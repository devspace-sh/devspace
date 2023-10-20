package sync

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/devspace/helper/remote"
	"github.com/loft-sh/devspace/helper/server/ignoreparser"
	"github.com/loft-sh/devspace/helper/util"
	"github.com/loft-sh/devspace/helper/util/crc32"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/restart"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	"google.golang.org/grpc"

	"github.com/loft-sh/notify"
	"github.com/loft-sh/utils/pkg/command"

	"github.com/bmatcuk/doublestar"
	"github.com/fujiwara/shapeio"
	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/expand"
)

var (
	TouchFile = ""
)

type upstream struct {
	events   chan notify.EventInfo
	symlinks map[string]*Symlink
	sync     *Sync

	reader io.ReadCloser
	writer io.WriteCloser
	client remote.UpstreamClient

	isBusy      bool
	isBusyMutex sync.Mutex

	eventBuffer      []notify.EventInfo
	eventBufferMutex sync.Mutex

	ignoreMatcher ignoreparser.IgnoreParser

	initialSyncCompletedMutex sync.Mutex
	initialSyncChanges        []string
	initialSyncCompleted      bool
	initialSyncTouchOnce      sync.Once

	conn *grpc.ClientConn
}

const (
	removeFilesBufferSize = 64
	largeFileSize         = 1024 * 1024 * 10
)

// newUpstream creates a new upstream handler with the given parameters
func newUpstream(reader io.ReadCloser, writer io.WriteCloser, sync *Sync) (*upstream, error) {
	var (
		clientReader io.Reader = reader
		clientWriter io.Writer = writer
	)

	// Apply limits if specified
	if sync.Options.DownstreamLimit > 0 {
		limitedReader := shapeio.NewReader(reader)
		limitedReader.SetRateLimit(float64(sync.Options.DownstreamLimit))
		clientReader = limitedReader
	}
	if sync.Options.UpstreamLimit > 0 {
		limitedWriter := shapeio.NewWriter(writer)
		limitedWriter.SetRateLimit(float64(sync.Options.UpstreamLimit))
		clientWriter = limitedWriter
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

	ignoreMatcher, err := ignoreparser.CompilePaths(excludePaths, sync.log)
	if err != nil {
		return nil, errors.Wrap(err, "compile paths")
	}

	return &upstream{
		events:      make(chan notify.EventInfo, 1000), // High buffer size so we don't miss any fsevents if there are a lot of changes
		eventBuffer: make([]notify.EventInfo, 0, 64),
		symlinks:    make(map[string]*Symlink),
		sync:        sync,
		isBusy:      true,

		reader: reader,
		writer: writer,
		client: remote.NewUpstreamClient(conn),

		ignoreMatcher: ignoreMatcher,

		conn: conn,
	}, nil
}

func (u *upstream) IsBusy() bool {
	u.isBusyMutex.Lock()
	defer u.isBusyMutex.Unlock()

	return len(u.events) > 0 || u.isBusy
}

func (u *upstream) IsInitialSyncing() bool {
	u.initialSyncCompletedMutex.Lock()
	defer u.initialSyncCompletedMutex.Unlock()

	return len(u.initialSyncChanges) > 0 || !u.initialSyncCompleted
}

func (u *upstream) startPing(doneChan chan struct{}) {
	go func() {
		for {
			select {
			case <-doneChan:
				return
			case <-time.After(time.Second * 15):
				if u.client != nil {
					ctx, cancel := context.WithTimeout(u.sync.ctx, time.Second*15)
					_, err := u.client.Ping(ctx, &remote.Empty{})
					cancel()
					if err != nil {
						u.sync.Stop(fmt.Errorf("ping connection: %v", err))
						return
					}
				}
			}
		}
	}()
}

func (u *upstream) startEventsLoop(doneChan chan struct{}) {
	go func() {
		for {
			select {
			case <-doneChan:
				return
			case event, ok := <-u.events:
				if !ok {
					return
				}

				// We need this loop to catch up if we got a lot of change events
				u.eventBufferMutex.Lock()
				u.eventBuffer = append(u.eventBuffer, event)
				for eventsLeft := true; eventsLeft; {
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
			changeTimer  time.Time
		)

		// gather changes
		for {
			select {
			case <-u.sync.ctx.Done():
				return nil
			case <-time.After(time.Millisecond * 600):
				break
			}

			// retrieve the newest events
			events := u.getEvents()
			if len(events) > 0 {
				fileInformation, err := u.getFileInformationFromEvent(events)
				if err != nil {
					return errors.Wrap(err, "get file information from event")
				}

				changes = append(changes, fileInformation...)
			}

			// start waiting timer
			if len(changes) > 0 && changeAmount == 0 {
				changeTimer = time.Now().Add(waitForMoreChangesTimeout)
			}

			// We gather changes till there are no more changes or
			// a certain amount of changes is reached
			if changeAmount > 0 && (time.Now().After(changeTimer) || len(changes) > 25000 || changeAmount == len(changes)) {
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

			err := u.execCommandsAfterInitialSync()
			if err != nil {
				return errors.Wrap(err, "exec after initial sync")
			}
		}

		// apply the changes
		err := u.applyChanges(changes)
		if err != nil {
			return errors.Wrap(err, "apply changes")
		}
	}
}

func (u *upstream) execCommandsAfterInitialSync() (err error) {
	u.initialSyncCompletedMutex.Lock()
	defer u.initialSyncCompletedMutex.Unlock()

	// make sure the touch file is there
	defer func() {
		if err == nil && u.initialSyncCompleted && (u.sync.Options.RestartContainer || u.sync.Options.StartContainer) {
			u.initialSyncTouchOnce.Do(func() {
				if u.sync.Options.Starter != nil {
					err = u.sync.Options.Starter.Done(u.startContainer)
				} else {
					err = u.startContainer()
				}
			})
		}
	}()

	if !u.initialSyncCompleted || len(u.initialSyncChanges) == 0 {
		return nil
	}

	changedFiles := u.initialSyncChanges
	u.initialSyncChanges = nil
	return u.execCommands(changedFiles)
}

func (u *upstream) startContainer() error {
	_, err := u.client.Execute(u.sync.ctx, &remote.Command{
		Cmd:  "touch",
		Args: []string{restart.TouchPath},
	})
	return err
}

func (u *upstream) execCommandsAfterApply(changedFiles []string) error {
	u.initialSyncCompletedMutex.Lock()
	defer u.initialSyncCompletedMutex.Unlock()

	if !u.initialSyncCompleted {
		u.initialSyncChanges = append(u.initialSyncChanges, changedFiles...)
		return nil
	}

	return u.execCommands(changedFiles)
}

func (u *upstream) execCommands(changedFiles []string) error {
	// execute exec commands
	for _, exec := range u.sync.Options.Exec {
		err := u.execCommand(exec, changedFiles)
		if err != nil {
			return err
		}
	}

	// execute batch command
	err := u.ExecuteBatchCommand()
	if err != nil {
		return err
	}

	// Restart container if needed
	return u.RestartContainer()
}

func (u *upstream) execCommand(exec latest.SyncExec, changedFiles []string) error {
	matched := ""
	if len(exec.OnChange) > 0 {
	Outer:
		for _, pattern := range exec.OnChange {
			pattern = path.Clean(pattern)
			for _, file := range changedFiles {
				if len(file) == 1 {
					continue
				}

				hasMatched, _ := doublestar.Match(pattern, file[1:])
				if hasMatched {
					matched = file[1:]
					break Outer
				}
			}
		}
		if matched == "" {
			return nil
		}
	}

	execCommand := exec.Command
	execArgs := exec.Args
	if u.sync.Options.ResolveCommand != nil {
		var err error
		execCommand, execArgs, err = u.sync.Options.ResolveCommand(execCommand, execArgs)
		if err != nil {
			return errors.Wrap(err, "resolve command")
		}
	}

	execCommandName := exec.Name
	if execCommandName == "" {
		execCommandName = command.FormatCommandName(execCommand, execArgs)
	}

	if exec.Local {
		if matched != "" {
			u.sync.log.Infof("Upstream - Execute command '%s' locally, because '%s' changed", execCommandName, matched)
		} else {
			u.sync.log.Infof("Upstream - Execute command '%s' locally", execCommandName)
		}

		// if args are nil we execute the command in a shell
		var (
			err error
			out = &bytes.Buffer{}
		)
		if exec.Args == nil {
			err = engine.ExecuteSimpleShellCommand(u.sync.ctx, u.sync.LocalPath, expand.ListEnviron(os.Environ()...), out, out, nil, execCommand)
		} else {
			err = command.Command(u.sync.ctx, u.sync.LocalPath, expand.ListEnviron(os.Environ()...), out, out, nil, execCommand, exec.Args...)
		}
		if err != nil {
			if exec.FailOnError {
				return fmt.Errorf("error executing command %s: %s %v", execCommandName, out.String(), err)
			}

			u.sync.log.Infof("Upstream - Error executing command: %s %v", out.String(), err)
		}

		u.sync.log.Infof("Upstream - Done executing command")
		return nil
	}

	ctx, cancel := context.WithTimeout(u.sync.ctx, time.Minute*10)
	defer cancel()

	cmd := execCommand
	args := execArgs
	if args == nil {
		args = []string{"-c", cmd}
		cmd = "sh"
	}

	if matched != "" {
		u.sync.log.Infof("Upstream - Execute command '%s', because '%s' changed", execCommandName, matched)
	} else {
		u.sync.log.Infof("Upstream - Execute command '%s'", execCommandName)
	}

	_, err := u.client.Execute(ctx, &remote.Command{
		Cmd:  cmd,
		Args: args,
		Once: exec.Once,
	})
	if err != nil {
		if exec.FailOnError {
			return errors.Wrap(err, "execute command")
		}

		u.sync.log.Infof("Upstream - Error executing command: %v", err)
	}
	u.sync.log.Infof("Upstream - Done executing command")
	return nil
}

func (u *upstream) getFileInformationFromEvent(events []notify.EventInfo) ([]*FileInformation, error) {
	u.sync.fileIndex.fileMapMutex.Lock()
	defer u.sync.fileIndex.fileMapMutex.Unlock()

	changes := make([]*FileInformation, 0, len(events))
	for _, event := range events {
		fileInfo, ok := event.(*FileInformation)

		// if the change is sent from the initial sync don't evaluate it
		if ok {
			changes = append(changes, fileInfo)
		} else {
			u.sync.log.Debugf("Upstream - Event from filesystem for %s", event.Path())

			// check if path is correct
			fullPath := event.Path()
			if !strings.HasPrefix(filepath.ToSlash(fullPath), filepath.ToSlash(u.sync.LocalPath)+"/") {
				u.sync.log.Infof("Upstream - unexpected upload path %s", fullPath)
				continue
			}

			// Determine what kind of change we got (Create or Remove)
			relativePath := getRelativeFromFullPath(fullPath, u.sync.LocalPath)
			newChanges, err := u.evaluateChange(relativePath, fullPath)
			if err != nil {
				return nil, errors.Wrap(err, "evaluate change")
			}

			changes = append(changes, newChanges...)
		}
	}

	return changes, nil
}

func (u *upstream) evaluateChange(relativePath, fullPath string) ([]*FileInformation, error) {
	stat, err := os.Stat(fullPath)
	if err != nil {
		// Remove symlinks
		u.RemoveSymlinks(fullPath)

		// Check if we should remove path remote
		if shouldRemoveRemote(relativePath, u.sync) {
			// New Remove Task
			return []*FileInformation{
				{
					Name: relativePath,
				},
			}, nil
		}

		return nil, nil
	}

	// Exclude changes on the upload exclude list
	if u.sync.uploadIgnoreMatcher != nil && u.sync.uploadIgnoreMatcher.Matches(relativePath, stat.IsDir()) {
		// Add to file map and prevent download if local file is newer than the remote one
		if u.sync.fileIndex.fileMap[relativePath] != nil && u.sync.fileIndex.fileMap[relativePath].Mtime < stat.ModTime().Unix() {
			// Add it to the fileMap
			u.sync.fileIndex.fileMap[relativePath] = &FileInformation{
				Name:        relativePath,
				Mtime:       stat.ModTime().Unix(),
				Mode:        stat.Mode(),
				Size:        stat.Size(),
				IsDirectory: stat.IsDir(),
			}
		}

		return nil, nil
	}

	// File / Folder exist -> Create File or Folder
	// if File / Folder does not exist, we create a new remove change
	// Check if symbolic link
	lstat, err := os.Lstat(fullPath)
	if err == nil && lstat.Mode()&os.ModeSymlink != 0 {
		_, symlinkExists := u.sync.upstream.symlinks[fullPath]

		// Add symlink to map
		stat, err = u.sync.upstream.AddSymlink(relativePath, fullPath)
		if err != nil {
			return nil, errors.Wrap(err, "add symlink")
		}
		if stat == nil {
			return nil, nil
		}

		// Only crawl if symlink wasn't there before and it is a directory
		if !symlinkExists && stat.IsDir() {
			// Crawl all linked files & folders
			err = u.symlinks[fullPath].Crawl()
			if err != nil {
				return nil, errors.Wrap(err, "crawl symlink")
			}
		}
	} else if err != nil {
		u.sync.log.Debugf("Error in lstat %s: %v", fullPath, err)
		return nil, nil
	} else if stat == nil {
		return nil, nil
	}

	fileInfo := &FileInformation{
		Name:           relativePath,
		Mtime:          stat.ModTime().Unix(),
		MtimeNano:      stat.ModTime().UnixNano(),
		Size:           stat.Size(),
		Mode:           stat.Mode(),
		IsDirectory:    stat.IsDir(),
		IsSymbolicLink: stat.Mode()&os.ModeSymlink != 0,
	}

	// should we upload the file?
	if shouldUpload(u.sync, fileInfo, u.sync.log) {
		// New Create Task
		return []*FileInformation{fileInfo}, nil
	} else if stat.IsDir() {
		// if the change is a directory we walk the directory for other potential changes
		files, err := os.ReadDir(fullPath)
		if err != nil {
			// Remove symlinks
			u.RemoveSymlinks(fullPath)

			// Check if we should remove path remote
			if shouldRemoveRemote(relativePath, u.sync) {
				// New Remove Task
				return []*FileInformation{
					{
						Name: relativePath,
					},
				}, nil
			}

			return nil, nil
		}

		changes := []*FileInformation{}
		for _, dirEntry := range files {
			f, err := dirEntry.Info()
			if err != nil {
				continue
			}

			newFullPath := filepath.Join(fullPath, f.Name())
			newRelativePath := path.Join(relativePath, f.Name())
			if fsutil.IsRecursiveSymlink(f, newFullPath) {
				continue
			}

			otherChanges, err := u.evaluateChange(newRelativePath, newFullPath)
			if err != nil {
				return nil, errors.Wrap(err, "evaluate change")
			}

			changes = append(changes, otherChanges...)
		}

		return changes, nil
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
		if u.sync.ignoreMatcher.Matches(relativePath, stat.IsDir()) {
			return nil, nil
		}
	}

	symlink, err := NewSymlink(u, absPath, targetPath, stat.IsDir(), u.sync.ignoreMatcher)
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
	u.sync.log.Debugf("Upstream - Start applying %d changes", len(changes))
	defer u.sync.log.Debugf("Upstream - Done applying changes")

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
	var writtenChanges map[string]*FileInformation
	if len(creates) > 0 {
		var err error
		writtenChanges, err = func() (map[string]*FileInformation, error) {
			u.sync.fileIndex.fileMapMutex.Lock()
			defer u.sync.fileIndex.fileMapMutex.Unlock()

			for i := 0; i < syncRetries; i++ {
				changes, err := u.applyCreates(creates)
				if err == nil {
					return changes, nil
				} else if i+1 >= syncRetries {
					return nil, errors.Wrap(err, "apply creates")
				} else if strings.Contains(err.Error(), "closed pipe") || strings.Contains(err.Error(), "transport is closing") || strings.Contains(err.Error(), "broken pipe") {
					return nil, errors.Wrap(err, "apply creates")
				}

				u.sync.log.Infof("Upstream - Retry upload because of error: %v", err)
				creates = u.updateUploadChanges(creates)
				if len(creates) == 0 {
					break
				}
			}

			return nil, nil
		}()
		if err != nil {
			return err
		}
	}

	changeAmount := len(removes) + len(writtenChanges)
	if changeAmount == 0 {
		return nil
	}

	u.sync.log.Infof("Upstream - Successfully processed %d change(s)", changeAmount)
	changeNames := make([]string, 0, changeAmount)
	for _, c := range removes {
		changeNames = append(changeNames, c.Name)
	}
	for n := range writtenChanges {
		changeNames = append(changeNames, n)
	}

	return u.execCommandsAfterApply(changeNames)
}

func (u *upstream) RestartContainer() error {
	if u.sync.Options.RestartContainer {
		u.sync.log.Info("Upstream - Restarting container")

		ctx, cancel := context.WithTimeout(u.sync.ctx, time.Minute*5)
		defer cancel()

		_, err := u.client.RestartContainer(ctx, &remote.Empty{})
		if err != nil {
			return errors.Wrap(err, "restart container")
		}
	}

	return nil
}

func (u *upstream) ExecuteBatchCommand() error {
	if u.sync.Options.UploadBatchCmd != "" {
		u.sync.log.Infof("Upstream - Execute command '%s %s'", u.sync.Options.UploadBatchCmd, strings.Join(u.sync.Options.UploadBatchArgs, " "))

		ctx, cancel := context.WithTimeout(u.sync.ctx, time.Minute*10)
		defer cancel()

		_, err := u.client.Execute(ctx, &remote.Command{
			Cmd:  u.sync.Options.UploadBatchCmd,
			Args: u.sync.Options.UploadBatchArgs,
		})
		if err != nil {
			return errors.Wrap(err, "execute command")
		}

		u.sync.log.Infof("Upstream - Done executing command")
	}

	return nil
}

func (u *upstream) updateUploadChanges(files []*FileInformation) []*FileInformation {
	newChanges := make([]*FileInformation, 0, len(files))
	for _, change := range files {
		if shouldUpload(u.sync, change, u.sync.log) {
			newChanges = append(newChanges, change)
		}
	}

	return newChanges
}

func (u *upstream) applyCreates(files []*FileInformation) (map[string]*FileInformation, error) {
	files, err := u.filterChanges(files)
	if err != nil {
		return nil, err
	} else if len(files) == 0 {
		return nil, nil
	}

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
	defer u.sync.log.Debugf("Upstream - Done Uploading")

	// Create a pipe for reading and writing
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()

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
		return nil, errors.Wrap(err, "upload archive")
	}

	// check if there was a compressing error
	err = <-errorChan
	if err != nil {
		return nil, errors.Wrap(err, "compress archive")
	}

	// finally update written files
	for _, element := range archiver.WrittenFiles() {
		u.sync.fileIndex.CreateDirInFileMap(path.Dir(element.Name))
		u.sync.fileIndex.fileMap[element.Name] = element
	}

	return archiver.WrittenFiles(), nil
}

func (u *upstream) filterChanges(files []*FileInformation) ([]*FileInformation, error) {
	alreadyUsed := map[string]bool{}
	newChanges := make([]*FileInformation, 0, len(files))
	needCheck := []*FileInformation{}

	// filter them first
	for _, f := range files {
		if f.Size > largeFileSize {
			u.sync.log.Debugf("Large file encountered at %s (%0.2f MB). Please try to avoid syncing large files as this will slow down DevSpace", u.getRelativeUpstreamPath(f.Name), float64(f.Size)/1024.0/1024.0)
		}

		if alreadyUsed[f.Name] {
			continue
		} else if f.IsDirectory || u.sync.fileIndex.fileMap[f.Name] == nil || u.sync.fileIndex.fileMap[f.Name].Size != f.Size {
			newChanges = append(newChanges, f)
			alreadyUsed[f.Name] = true
			continue
		} else if f.Size == 0 {
			alreadyUsed[f.Name] = true
			continue
		}

		alreadyUsed[f.Name] = true
		needCheck = append(needCheck, f)
	}

	// now compare crc32 hashes
	if len(needCheck) > 0 {
		u.sync.log.Debugf("Start hashing %d files", len(needCheck))
		defer u.sync.log.Debugf("Done hashing %d files", len(needCheck))

		// cancel after 10 minutes
		ctx, cancel := context.WithTimeout(u.sync.ctx, time.Minute*30)
		defer cancel()

		// create done chan
		done := make(chan error)

		// start remote hashing
		remoteChecksums := make([]uint32, 0, len(needCheck))
		localChecksums := make([]uint32, 0, len(needCheck))
		go func() {
			// send 1000 each time
			batchSize := 1000
			for i := 0; i < len(needCheck); i += batchSize {
				batch := make([]*remote.TouchPath, 0, batchSize)
				for j := 0; j < batchSize; j++ {
					if i+j >= len(needCheck) {
						break
					}

					change := needCheck[i+j]
					u.sync.fileIndex.fileMap[change.Name].Mtime = change.Mtime
					touchPath := &remote.TouchPath{
						Path:      change.Name,
						MtimeUnix: change.Mtime,
					}
					if !equalFilePermissions(u.sync.fileIndex.fileMap[change.Name].Mode, change.Mode) {
						u.sync.fileIndex.fileMap[change.Name].Mode = change.Mode
						touchPath.Mode = uint32(change.Mode)
					}

					batch = append(batch, touchPath)
				}

				// ask remote for checksums
				checksums, err := u.client.Checksums(ctx, &remote.TouchPaths{Paths: batch})
				if err != nil {
					done <- err
					return
				} else if checksums == nil {
					done <- fmt.Errorf("unexpected checksum response")
					return
				} else if len(checksums.Checksums) != len(batch) {
					done <- fmt.Errorf("unexpected checksum size %d != %d", len(checksums.Checksums), len(batch))
					return
				}

				remoteChecksums = append(remoteChecksums, checksums.Checksums...)
			}

			done <- nil
		}()

		// start local hashing
		for _, c := range needCheck {
			// Just remove everything inside and ignore any errors
			absolutePath := path.Join(u.sync.LocalPath, c.Name)
			checksum, err := crc32.Checksum(absolutePath)
			if err != nil && !os.IsNotExist(err) {
				u.sync.log.Infof("Error hashing file %s: %v", c.Name, err)
			}

			localChecksums = append(localChecksums, checksum)
		}

		// wait for remote
		err := <-done
		if err != nil {
			return nil, errors.Wrap(err, "hashing remote files")
		} else if len(remoteChecksums) != len(localChecksums) {
			return nil, fmt.Errorf("unexpected checksum size %d != %d", len(remoteChecksums), len(localChecksums))
		}

		// compare checksums
		for i := range remoteChecksums {
			if remoteChecksums[i] != 0 && remoteChecksums[i] == localChecksums[i] {
				continue
			}

			newChanges = append(newChanges, needCheck[i])
		}
	}

	return newChanges, nil
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

func (u *upstream) uploadArchive(reader io.ReadCloser) error {
	defer reader.Close()

	// cancel after 1 hour
	ctx, cancel := context.WithTimeout(u.sync.ctx, time.Hour)
	defer cancel()

	// Create upload client
	uploadClient, err := u.client.Upload(ctx)
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

	ctx, cancel := context.WithTimeout(u.sync.ctx, time.Minute*30)
	defer cancel()

	u.sync.log.Infof("Upstream - Handling %d removes", len(files))
	defer u.sync.log.Debugf("Upstream - Done Handling removes")
	fileMap := u.sync.fileIndex.fileMap

	removeClient, err := u.client.Remove(ctx)
	if err != nil {
		return errors.Wrap(err, "remove client")
	}

	sendFiles := make([]string, 0, removeFilesBufferSize)
	for _, file := range files {
		u.sync.log.Infof("Upstream - Remove '%s'", u.getRelativeUpstreamPath(file.Name))
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

	return uploadPath[1:]
}
