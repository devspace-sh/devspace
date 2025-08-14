package sync

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/loft-sh/devspace/helper/remote"
	"github.com/loft-sh/devspace/helper/server/ignoreparser"
	"github.com/loft-sh/devspace/helper/util"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	"google.golang.org/grpc"

	"github.com/fujiwara/shapeio"
	"github.com/pkg/errors"
)

type downstream struct {
	sync *Sync

	reader io.ReadCloser
	writer io.WriteCloser
	client remote.DownstreamClient

	ignoreMatcher ignoreparser.IgnoreParser
	conn          *grpc.ClientConn

	unarchiver *Unarchiver
}

const downloadFilesBufferSize = 64

// newDownstream creates a new downstream handler with the given parameters
func newDownstream(reader io.ReadCloser, writer io.WriteCloser, sync *Sync) (*downstream, error) {
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

	// Create client connection
	conn, err := util.NewClientConnection(clientReader, clientWriter)
	if err != nil {
		return nil, errors.Wrap(err, "new client connection")
	}

	// Create download exclude paths
	ignoreMatcher, err := ignoreparser.CompilePaths(sync.Options.DownloadExcludePaths, sync.log)
	if err != nil {
		return nil, errors.Wrap(err, "compile paths")
	}

	return &downstream{
		sync:          sync,
		reader:        reader,
		writer:        writer,
		client:        remote.NewDownstreamClient(conn),
		ignoreMatcher: ignoreMatcher,
		unarchiver:    NewUnarchiver(sync, false, sync.log),
		conn:          conn,
	}, nil
}

func (d *downstream) populateFileMap() error {
	d.sync.fileIndex.fileMapMutex.Lock()
	defer d.sync.fileIndex.fileMapMutex.Unlock()

	changes, err := d.collectChanges(true)
	if err != nil {
		return errors.Wrap(err, "collect changes")
	}

	for _, element := range changes {
		if d.sync.fileIndex.fileMap[element.Path] == nil {
			d.sync.fileIndex.fileMap[element.Path] = parseFileInformation(element)
		}
	}

	return nil
}

func (d *downstream) collectChanges(skipIgnore bool) ([]*remote.Change, error) {
	d.sync.log.Debugf("Downstream - Start collecting changes")
	defer d.sync.log.Debugf("Downstream - Done collecting changes")

	changes := make([]*remote.Change, 0, 128)
	ctx, cancel := context.WithTimeout(d.sync.ctx, time.Minute*30)
	defer cancel()

	// Create a change client and collect all changes
	changesClient, err := d.client.Changes(ctx, &remote.Empty{})
	if err != nil {
		return nil, errors.Wrap(err, "start retrieving changes")
	}

	for {
		changeChunk, err := changesClient.Recv()
		if changeChunk != nil {
			for _, change := range changeChunk.Changes {
				if !skipIgnore && d.ignoreMatcher != nil && d.ignoreMatcher.Matches(change.Path, change.IsDir) {
					continue
				}
				if !d.shouldKeep(change) {
					continue
				}

				changes = append(changes, change)
			}
		}

		if err == io.EOF {
			break
		} else if err != nil {
			return nil, errors.Wrap(err, "recv change")
		}
	}

	return changes, nil
}

func (d *downstream) startPing(doneChan chan struct{}) {
	go func() {
		for {
			select {
			case <-doneChan:
				return
			case <-time.After(time.Second * 15):
				if d.client != nil {
					ctx, cancel := context.WithTimeout(d.sync.ctx, time.Second*20)
					_, err := d.client.Ping(ctx, &remote.Empty{})
					cancel()
					if err != nil {
						d.sync.Stop(fmt.Errorf("ping connection: %v", err))
						return
					}
				}
			}
		}
	}()
}

func (d *downstream) mainLoop() error {
	lastAmountChanges := int64(0)
	recheckInterval := 1700
	if !d.sync.Options.Polling {
		recheckInterval = 500
	}

	var (
		changeTimer time.Time
	)
	for {
		select {
		case <-d.sync.ctx.Done():
			return nil
		case <-time.After(time.Duration(recheckInterval) * time.Millisecond):
			break
		}

		// Check for changes remotely
		ctx, cancel := context.WithTimeout(d.sync.ctx, time.Minute*10)
		changeAmount, err := d.client.ChangesCount(ctx, &remote.Empty{})
		cancel()
		if err != nil {
			return errors.Wrap(err, "count changes")
		}

		// start waiting timer
		if changeAmount.Amount > 0 && lastAmountChanges == 0 {
			changeTimer = time.Now().Add(waitForMoreChangesTimeout)
		}

		// Compare change amount
		if lastAmountChanges > 0 && (time.Now().After(changeTimer) || changeAmount.Amount > 25000 || changeAmount.Amount == lastAmountChanges) {
			d.sync.fileIndex.fileMapMutex.Lock()
			changes, err := d.collectChanges(false)
			d.sync.fileIndex.fileMapMutex.Unlock()
			if err != nil {
				return errors.Wrap(err, "collect changes")
			}

			err = d.applyChanges(changes, false)
			if err != nil {
				return errors.Wrap(err, "apply changes")
			}

			lastAmountChanges = 0
			changeTimer = time.Time{}
		} else {
			lastAmountChanges = changeAmount.Amount
		}
	}
}

func (d *downstream) shouldKeep(change *remote.Change) bool {
	// Is a delete change?
	if change.ChangeType == remote.ChangeType_DELETE {
		return shouldRemoveLocal(filepath.Join(d.sync.LocalPath, change.Path), parseFileInformation(change), d.sync, false)
	}

	// Exclude symlinks
	// if fileInformation.IsSymbolicLink {
	// Add them to the fileMap though
	// d.config.fileIndex.fileMap[fileInformation.Name] = fileInformation
	// }

	// Should we download the file / folder?
	return shouldDownload(change, d.sync)
}

func (d *downstream) applyChanges(changes []*remote.Change, force bool) error {
	d.sync.log.Debugf("Downstream - Start applying %d changes", len(changes))
	defer d.sync.log.Debugf("Downstream - Done applying changes")

	var (
		download = make([]*remote.Change, 0, len(changes)/2)
		remove   = make([]*remote.Change, 0, len(changes)/2)
	)

	// Skip if there are no changes
	if len(changes) == 0 {
		return nil
	}

	// determine what to delete and what to download
	for _, change := range changes {
		if change.ChangeType == remote.ChangeType_DELETE {
			remove = append(remove, change)
		} else {
			download = append(download, change)
		}
	}

	// Remove all files and folders that should be deleted first and we ignore errors
	d.remove(remove, force)

	// Extract downloaded archive
	if len(download) > 0 {
		for i := 0; i < syncRetries; i++ {
			err := d.initDownload(download)
			if err == nil {
				break
			} else if i+1 >= syncRetries {
				return err
			}

			d.sync.log.Infof("Downstream - Retry download because of error: %v", err)

			download = d.updateDownloadChanges(download)
			if len(download) == 0 {
				break
			}
		}
	}

	d.sync.log.Infof("Downstream - Successfully processed %d change(s)", len(changes))
	return nil
}

func (d *downstream) updateDownloadChanges(download []*remote.Change) []*remote.Change {
	d.sync.fileIndex.fileMapMutex.Lock()
	defer d.sync.fileIndex.fileMapMutex.Unlock()

	newChanges := make([]*remote.Change, 0, len(download))
	for _, change := range download {
		if d.shouldKeep(change) {
			newChanges = append(newChanges, change)
		}
	}

	return newChanges
}

func (d *downstream) initDownload(download []*remote.Change) error {
	reader, writer := io.Pipe()

	defer reader.Close()
	defer writer.Close()

	errorChan := make(chan error)
	go func() {
		errorChan <- d.downloadFiles(writer, download)
	}()

	// Untaring all downloaded files to the right location
	// this can be a lengthy process when we downloaded a lot of files
	err := d.unarchiver.Untar(reader, d.sync.LocalPath)
	if err != nil {
		return errors.Wrap(err, "untar files")
	}

	return <-errorChan
}

// downloadFiles downloads the given files from the remote server and writes the contents into the given writer
func (d *downstream) downloadFiles(writer io.WriteCloser, changes []*remote.Change) error {
	defer writer.Close()

	// cancel after 1 hour
	ctx, cancel := context.WithTimeout(d.sync.ctx, time.Hour)
	defer cancel()

	// Print log message
	if len(changes) <= 3 || d.sync.Options.Verbose {
		for _, element := range changes {
			d.sync.log.Infof("Downstream - Download file '.%s', uncompressed: ~%0.2f KB", element.Path, float64(element.Size)/1024.0)
		}
	} else if len(changes) > 3 {
		filesize := int64(0)
		for _, v := range changes {
			filesize += v.Size
		}

		d.sync.log.Infof("Downstream - Download %d file(s) (Uncompressed: ~%0.2f KB)", len(changes), float64(filesize)/1024.0)
	}

	// Create new download client
	downloadClient, err := d.client.Download(ctx)
	if err != nil {
		return errors.Wrap(err, "download files")
	}

	// Send files to download
	downloadFiles := make([]string, 0, downloadFilesBufferSize)
	for _, change := range changes {
		downloadFiles = append(downloadFiles, change.Path)

		if len(downloadFiles) >= downloadFilesBufferSize {
			err = downloadClient.Send(&remote.Paths{
				Paths: downloadFiles,
			})
			if err != nil {
				return errors.Wrap(err, "send path")
			}

			downloadFiles = make([]string, 0, downloadFilesBufferSize)
		}
	}

	if len(downloadFiles) >= 0 {
		err = downloadClient.Send(&remote.Paths{
			Paths: downloadFiles,
		})
		if err != nil {
			return errors.Wrap(err, "send path")
		}
	}

	// We finish sending and start receiving the tar
	err = downloadClient.CloseSend()
	if err != nil {
		return errors.Wrap(err, "close send")
	}

	// Download tar archive into the writer
	for {
		chunk, err := downloadClient.Recv()
		if chunk != nil {
			_, err := writer.Write(chunk.Content)
			if err != nil {
				// this means the tar is done already, so we just exit here
				if strings.Contains(err.Error(), "io: read/write on closed pipe") {
					return nil
				}

				return errors.Wrap(err, "write chunk")
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "download recv")
		}
	}

	return nil
}

func (d *downstream) remove(remove []*remote.Change, force bool) {
	d.sync.fileIndex.fileMapMutex.Lock()
	defer d.sync.fileIndex.fileMapMutex.Unlock()

	fileMap := d.sync.fileIndex.fileMap

	// Remove Files & Folders
	numRemoveFiles := len(remove)
	if numRemoveFiles > 3 {
		d.sync.log.Infof("Downstream - Remove %d files", numRemoveFiles)
	}

	for _, change := range remove {
		absFilepath := filepath.Join(d.sync.LocalPath, change.Path)
		if shouldRemoveLocal(absFilepath, parseFileInformation(change), d.sync, force) {
			if numRemoveFiles <= 3 || d.sync.Options.Verbose {
				d.sync.log.Infof("Downstream - Remove '.%s'", change.Path)
			}

			if change.IsDir {
				d.deleteSafeRecursive(change.Path, remove, force)
			} else {
				err := os.Remove(absFilepath)
				if err != nil {
					if !os.IsNotExist(err) {
						d.sync.log.Infof("Downstream - Skip file delete '.%s': %v", change.Path, err)
					}
				}
			}
		}

		delete(fileMap, change.Path)
	}
}

func (d *downstream) deleteSafeRecursive(relativePath string, deleteChanges []*remote.Change, force bool) {
	absolutePath := filepath.Join(d.sync.LocalPath, relativePath)
	relativePath = getRelativeFromFullPath(absolutePath, d.sync.LocalPath)

	// Check if path is in delete changes
	found := false
	for _, remove := range deleteChanges {
		if remove.Path == relativePath {
			found = true
		}
	}

	// We don't delete the folder or the contents if we haven't tracked it
	if !force {
		if d.sync.fileIndex.fileMap[relativePath] == nil || !found {
			d.sync.log.Infof("Downstream - Skip delete directory '.%s'", relativePath)
			return
		}
	}

	// Delete directory from fileMap
	defer delete(d.sync.fileIndex.fileMap, relativePath)
	files, err := os.ReadDir(absolutePath)
	if err != nil {
		return
	}

	// Loop over directory contents and check if we should delete the contents
	for _, dirEntry := range files {
		f, err := dirEntry.Info()
		if err != nil {
			continue
		}

		if fsutil.IsRecursiveSymlink(f, path.Join(relativePath, f.Name())) {
			continue
		}

		childRelativePath := filepath.ToSlash(filepath.Join(relativePath, f.Name()))
		childAbsFilepath := filepath.Join(d.sync.LocalPath, childRelativePath)
		if shouldRemoveLocal(childAbsFilepath, d.sync.fileIndex.fileMap[childRelativePath], d.sync, force) {
			if f.IsDir() {
				d.deleteSafeRecursive(childRelativePath, deleteChanges, force)
			} else {
				err = os.Remove(childAbsFilepath)
				if err != nil {
					d.sync.log.Infof("Downstream - Skip file delete '.%s': %v", relativePath, err)
				}
			}
		} else {
			d.sync.log.Infof("Downstream - Skip delete '.%s'", relativePath)
		}

		delete(d.sync.fileIndex.fileMap, childRelativePath)
	}

	// This will not remove the directory if there is still a file or directory in it
	err = os.Remove(absolutePath)
	if err != nil {
		d.sync.log.Infof("Downstream - Skip delete directory '.%s', because %s", relativePath, err.Error())
	}
}
