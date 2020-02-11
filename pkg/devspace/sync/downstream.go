package sync

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/juju/ratelimit"
	"github.com/pkg/errors"

	"github.com/devspace-cloud/devspace/sync/remote"
	"github.com/devspace-cloud/devspace/sync/util"
)

type downstream struct {
	interrupt chan bool
	sync      *Sync

	reader io.ReadCloser
	writer io.WriteCloser
	client remote.DownstreamClient
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
		clientReader = ratelimit.Reader(reader, ratelimit.NewBucketWithRate(float64(sync.Options.DownstreamLimit), sync.Options.DownstreamLimit))
	}
	if sync.Options.UpstreamLimit > 0 {
		clientWriter = ratelimit.Writer(writer, ratelimit.NewBucketWithRate(float64(sync.Options.UpstreamLimit), sync.Options.UpstreamLimit))
	}

	// Create client connection
	conn, err := util.NewClientConnection(clientReader, clientWriter)
	if err != nil {
		return nil, errors.Wrap(err, "new client connection")
	}

	return &downstream{
		interrupt: make(chan bool, 1),
		sync:      sync,
		reader:    reader,
		writer:    writer,
		client:    remote.NewDownstreamClient(conn),
	}, nil
}

func (d *downstream) populateFileMap() error {
	d.sync.fileIndex.fileMapMutex.Lock()
	defer d.sync.fileIndex.fileMapMutex.Unlock()

	changes, err := d.collectChanges()
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

func (d *downstream) collectChanges() ([]*remote.Change, error) {
	changes := make([]*remote.Change, 0, 128)

	// Create a change client and collect all changes
	changesClient, err := d.client.Changes(context.Background(), &remote.Empty{})
	if err != nil {
		return nil, errors.Wrap(err, "start retrieving changes")
	}

	for {
		changeChunk, err := changesClient.Recv()
		if changeChunk != nil {
			for _, change := range changeChunk.Changes {
				if d.shouldKeep(change) {
					changes = append(changes, change)
				}
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

func (d *downstream) mainLoop() error {
	lastAmountChanges := int64(0)

	for {
		// Check for changes remotely
		changeAmount, err := d.client.ChangesCount(context.Background(), &remote.Empty{})
		if err != nil {
			return errors.Wrap(err, "count changes")
		}

		// Compare change amount
		if lastAmountChanges > 0 && changeAmount.Amount == lastAmountChanges {
			d.sync.fileIndex.fileMapMutex.Lock()
			changes, err := d.collectChanges()
			d.sync.fileIndex.fileMapMutex.Unlock()
			if err != nil {
				return errors.Wrap(err, "collect changes")
			}

			err = d.applyChanges(changes)
			if err != nil {
				return errors.Wrap(err, "apply changes")
			}
		}

		select {
		case <-d.interrupt:
			return nil
		case <-time.After(1700 * time.Millisecond):
			break
		}

		lastAmountChanges = changeAmount.Amount
	}
}

func (d *downstream) shouldKeep(change *remote.Change) bool {
	// Is a delete change?
	if change.ChangeType == remote.ChangeType_DELETE {
		return shouldRemoveLocal(filepath.Join(d.sync.LocalPath, change.Path), parseFileInformation(change), d.sync)
	}

	// Exclude symlinks
	// if fileInformation.IsSymbolicLink {
	// Add them to the fileMap though
	// d.config.fileIndex.fileMap[fileInformation.Name] = fileInformation
	// }

	// Should we download the file / folder?
	return shouldDownload(change, d.sync)
}

func (d *downstream) applyChanges(changes []*remote.Change) error {
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
	d.remove(remove)

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
	reader, writer, err := os.Pipe()
	if err != nil {
		return errors.Wrap(err, "create pipe")
	}

	defer reader.Close()
	defer writer.Close()

	errorChan := make(chan error)
	go func() {
		errorChan <- d.downloadFiles(writer, download)
	}()

	// Untaring all downloaded files to the right location
	// this can be a lengthy process when we downloaded a lot of files
	err = untarAll(reader, d.sync.LocalPath, "", d.sync)
	if err != nil {
		return errors.Wrap(err, "untar files")
	}

	return <-errorChan
}

// downloadFiles downloads the given files from the remote server and writes the contents into the given writer
func (d *downstream) downloadFiles(writer io.WriteCloser, changes []*remote.Change) error {
	defer writer.Close()

	// Print log message
	if len(changes) <= 3 || d.sync.Options.Verbose {
		for _, element := range changes {
			d.sync.log.Infof("Downstream - Download file %s, size: %d", element.Path, element.Size)
		}
	} else if len(changes) > 3 {
		filesize := int64(0)
		for _, v := range changes {
			filesize += v.Size
		}

		d.sync.log.Infof("Downstream - Download %d files (size: %d)", len(changes), filesize)
	}

	// Create new download client
	downloadClient, err := d.client.Download(context.Background())
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

func (d *downstream) remove(remove []*remote.Change) {
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
		if shouldRemoveLocal(absFilepath, parseFileInformation(change), d.sync) {
			if numRemoveFiles <= 3 || d.sync.Options.Verbose {
				d.sync.log.Infof("Downstream - Remove %s", change.Path)
			}

			if change.IsDir {
				d.deleteSafeRecursive(change.Path, remove)
			} else {
				err := os.Remove(absFilepath)
				if err != nil {
					if os.IsNotExist(err) == false {
						d.sync.log.Infof("Downstream - Skip file delete %s: %v", change.Path, err)
					}
				}
			}
		}

		delete(fileMap, change.Path)
	}
}

func (d *downstream) deleteSafeRecursive(relativePath string, deleteChanges []*remote.Change) {
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
	if d.sync.fileIndex.fileMap[relativePath] == nil || found == false {
		d.sync.log.Infof("Downstream - Skip delete directory %s\n", relativePath)
		return
	}

	// Delete directory from fileMap
	defer delete(d.sync.fileIndex.fileMap, relativePath)
	files, err := ioutil.ReadDir(absolutePath)
	if err != nil {
		return
	}

	// Loop over directory contents and check if we should delete the contents
	for _, f := range files {
		childRelativePath := filepath.ToSlash(filepath.Join(relativePath, f.Name()))
		childAbsFilepath := filepath.Join(d.sync.LocalPath, childRelativePath)
		if shouldRemoveLocal(childAbsFilepath, d.sync.fileIndex.fileMap[childRelativePath], d.sync) {
			if f.IsDir() {
				d.deleteSafeRecursive(childRelativePath, deleteChanges)
			} else {
				err = os.Remove(childAbsFilepath)
				if err != nil {
					d.sync.log.Infof("Downstream - Skip file delete %s: %v", relativePath, err)
				}
			}
		} else {
			d.sync.log.Infof("Downstream - Skip delete %s", relativePath)
		}

		delete(d.sync.fileIndex.fileMap, childRelativePath)
	}

	// This will not remove the directory if there is still a file or directory in it
	err = os.Remove(absolutePath)
	if err != nil {
		d.sync.log.Infof("Downstream - Skip delete directory %s, because %s", relativePath, err.Error())
	}
}
