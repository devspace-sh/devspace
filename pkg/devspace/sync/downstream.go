package sync

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/juju/ratelimit"

	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
)

type downstream struct {
	interrupt chan bool
	config    *SyncConfig

	stdinPipe  io.WriteCloser
	stdoutPipe io.ReadCloser
	stderrPipe io.ReadCloser
}

func (d *downstream) start() error {
	d.interrupt = make(chan bool, 1)

	err := d.startShell()
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (d *downstream) startShell() error {
	if d.config.testing == false {
		stdinReader, stdinWriter, _ := os.Pipe()
		stdoutReader, stdoutWriter, _ := os.Pipe()
		stderrReader, stderrWriter, _ := os.Pipe()

		go func() {
			err := kubectl.ExecStream(d.config.Kubectl, d.config.Pod, d.config.Container.Name, []string{"sh"}, false, stdinReader, stdoutWriter, stderrWriter)
			if err != nil {
				d.config.Error(errors.Trace(err))
			}
		}()

		d.stdinPipe = stdinWriter
		d.stdoutPipe = stdoutReader
		d.stderrPipe = stderrReader
	} else {
		var err error

		cmd := exec.Command("sh")

		d.stdinPipe, err = cmd.StdinPipe()
		if err != nil {
			return err
		}

		d.stdoutPipe, err = cmd.StdoutPipe()
		if err != nil {
			return err
		}

		d.stderrPipe, err = cmd.StderrPipe()
		if err != nil {
			return err
		}

		err = cmd.Start()
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *downstream) populateFileMap() error {
	createFiles, err := d.collectChanges(nil)
	if err != nil {
		return errors.Trace(err)
	}

	d.config.fileIndex.fileMapMutex.Lock()
	defer d.config.fileIndex.fileMapMutex.Unlock()

	for _, element := range createFiles {
		if d.config.fileIndex.fileMap[element.Name] == nil {
			d.config.fileIndex.fileMap[element.Name] = element
		}
	}

	return nil
}

func (d *downstream) mainLoop() error {
	lastAmountChanges := 0

	for {
		removeFiles := d.cloneFileMap()

		// Check for changes remotely
		createFiles, err := d.collectChanges(removeFiles)
		if err != nil {
			return errors.Trace(err)
		}

		amountChanges := len(createFiles) + len(removeFiles)
		if lastAmountChanges > 0 && amountChanges == lastAmountChanges {
			err = d.applyChanges(createFiles, removeFiles)
			if err != nil {
				return errors.Trace(err)
			}
		}

		select {
		case <-d.interrupt:
			return nil
		case <-time.After(1300 * time.Millisecond):
			break
		}

		lastAmountChanges = len(createFiles) + len(removeFiles)
	}
}

func (d *downstream) cloneFileMap() map[string]*fileInformation {
	d.config.fileIndex.fileMapMutex.Lock()
	defer d.config.fileIndex.fileMapMutex.Unlock()

	mapClone := make(map[string]*fileInformation)

	for key, value := range d.config.fileIndex.fileMap {
		if value.IsSymbolicLink {
			continue
		}

		mapClone[key] = &fileInformation{
			Name:        value.Name,
			Size:        value.Size,
			Mtime:       value.Mtime,
			IsDirectory: value.IsDirectory,
		}
	}

	return mapClone
}

func (d *downstream) collectChanges(removeFiles map[string]*fileInformation) ([]*fileInformation, error) {
	createFiles := make([]*fileInformation, 0, 128)
	destPathFound := false

	// Write find command to stdin pipe
	cmd := getFindCommand(d.config.DestPath)
	_, err := d.stdinPipe.Write([]byte(cmd))
	if err != nil {
		return nil, errors.Trace(err)
	}

	buf := make([]byte, 0, 512)
	overlap := ""
	done := false

	var downloadReader io.Reader = d.stdoutPipe
	if d.config.DownstreamLimit > 0 {
		downloadReader = ratelimit.Reader(d.stdoutPipe, ratelimit.NewBucketWithRate(float64(d.config.DownstreamLimit), d.config.DownstreamLimit))
	}

	for done == false {
		n, err := downloadReader.Read(buf[:cap(buf)])
		buf = buf[:n]

		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				return nil, errors.Trace(fmt.Errorf("\n[Downstream] Stream closed unexpectedly"))
			}

			return nil, errors.Trace(err)
		}

		// Error reading from stdout
		if err != nil && err != io.EOF {
			return nil, errors.Trace(err)
		}

		done, overlap, err = d.parseLines(string(buf), overlap, &createFiles, removeFiles, &destPathFound)
		if err != nil {
			if _, ok := err.(parsingError); ok {
				time.Sleep(time.Second * 4)
				return d.collectChanges(removeFiles)
			}

			// No trace here because it could be a parsing error
			return nil, err
		}
	}

	if destPathFound == false {
		return nil, errors.New("DestPath not found, find command did not execute correctly")
	}

	return createFiles, nil
}

func (d *downstream) parseLines(buffer, overlap string, createFiles *[]*fileInformation, removeFiles map[string]*fileInformation, destPathFound *bool) (bool, string, error) {
	lines := strings.Split(buffer, "\n")

	for index, element := range lines {
		line := ""

		if index == 0 {
			if len(lines) > 1 {
				line = overlap + element
			} else {
				overlap += element
			}
		} else if index == len(lines)-1 {
			overlap = element
		} else {
			line = element
		}

		if line == EndAck || overlap == EndAck {
			return true, overlap, nil
		} else if line == ErrorAck || overlap == ErrorAck {
			return true, "", parsingError{
				msg: "Parsing Error",
			}
		} else if line != "" {
			destPath, err := d.evaluateFile(line, createFiles, removeFiles)
			if destPath {
				*destPathFound = destPath
			}

			if err != nil {
				return true, "", errors.Trace(err)
			}
		}
	}

	return false, overlap, nil
}

func (d *downstream) evaluateFile(fileline string, createFiles *[]*fileInformation, removeFiles map[string]*fileInformation) (bool, error) {
	d.config.fileIndex.fileMapMutex.Lock()
	defer d.config.fileIndex.fileMapMutex.Unlock()

	fileInformation, err := parseFileInformation(fileline, d.config.DestPath)

	// Error parsing line
	if err != nil {
		return false, errors.Trace(err)
	}

	// No file found
	if fileInformation == nil {
		return true, nil
	}

	// File found don't delete it
	delete(removeFiles, fileInformation.Name)

	// Update mode, gid & uid if exists
	if d.config.fileIndex.fileMap[fileInformation.Name] != nil {
		d.config.fileIndex.fileMap[fileInformation.Name].RemoteMode = fileInformation.RemoteMode
		d.config.fileIndex.fileMap[fileInformation.Name].RemoteGID = fileInformation.RemoteGID
		d.config.fileIndex.fileMap[fileInformation.Name].RemoteUID = fileInformation.RemoteUID
	}

	// Exclude symlinks
	if fileInformation.IsSymbolicLink {
		// Add them to the fileMap though
		d.config.fileIndex.fileMap[fileInformation.Name] = fileInformation
	}

	// Should we download the file / folder?
	if shouldDownload(fileInformation, d.config) {
		*createFiles = append(*createFiles, fileInformation)
	}

	return false, nil
}

func (d *downstream) applyChanges(createFiles []*fileInformation, removeFiles map[string]*fileInformation) error {
	var err error

	downloadFiles := make([]*fileInformation, 0, int(len(createFiles)/2))
	createFolders := make([]*fileInformation, 0, int(len(createFiles)/2))
	tempDownloadpath := ""

	// Determine folder creates and file creates and separate them
	for _, element := range createFiles {
		if element.IsDirectory {
			createFolders = append(createFolders, element)
		} else {
			downloadFiles = append(downloadFiles, element)
		}
	}

	// Download files first without locking the fileMap so upstream has more time to process other changes
	if len(downloadFiles) > 0 {
		tempDownloadpath, err = d.downloadFiles(downloadFiles)

		if err != nil {
			return errors.Trace(err)
		}

		defer os.Remove(tempDownloadpath)
	}

	d.removeFilesAndFolders(removeFiles)
	d.createFolders(createFolders)

	if len(downloadFiles) > 0 {
		f, err := os.Open(tempDownloadpath)
		if err != nil {
			return errors.Trace(err)
		}

		defer f.Close()

		// Untaring all downloaded files to the right location
		// this can be a lengthy process when we downloaded a lot of files
		err = untarAll(f, d.config.WatchPath, d.config.DestPath, d.config)
		if err != nil {
			return errors.Trace(err)
		}
	}

	d.config.Logf("[Downstream] Successfully processed %d change(s)", len(createFiles)+len(removeFiles))
	return nil
}

func (d *downstream) downloadFiles(files []*fileInformation) (string, error) {
	var buffer bytes.Buffer
	lenFiles := len(files)

	if lenFiles > 3 {
		filesize := int64(0)

		for _, v := range files {
			filesize += v.Size
		}

		d.config.Logf("[Downstream] Download %d files (size: %d)", lenFiles, filesize)
	}

	// Each file is represented in one line
	for _, element := range files {
		if lenFiles <= 3 || d.config.Verbose {
			d.config.Logf("[Downstream] Download file %s, size: %d", element.Name, element.Size)
		}

		buffer.WriteString(d.config.DestPath + element.Name)
		buffer.WriteString("\n")
	}

	filenames := buffer.String()

	// TODO: Implement timeout to prevent potential endless loop
	cmd := "fileSize=" + strconv.Itoa(len(filenames)) + `;
					tmpFileInput="/tmp/devspace-downstream-input";
					tmpFileOutput="/tmp/devspace-downstream-output";
					mkdir -p /tmp;

					pid=$$;
					cat </proc/$pid/fd/0 >"$tmpFileInput" &
					ddPid=$!;

					echo "` + StartAck + `";

					while true; do
							bytesRead=$(stat -c "%s" "$tmpFileInput" 2>/dev/null || printf "0");
						
							if [ "$bytesRead" = "$fileSize" ]; then
									kill $ddPid;
									break;
							fi;

							sleep 0.1;
					done;
					tar -czf "$tmpFileOutput" -T "$tmpFileInput" 2>/tmp/devspace-downstream-error;
					(>&2 echo "` + StartAck + `");
					(>&2 echo $(stat -c "%s" "$tmpFileOutput"));
					(>&2 echo "` + EndAck + `");
					cat "$tmpFileOutput";
		` // We need that extra new line, otherwise the command is not executed properly

	// Write command to stdin
	_, err := d.stdinPipe.Write([]byte(cmd))
	if err != nil {
		return "", errors.Trace(err)
	}

	// Wait till remote is ready to receive filenames
	err = waitTill(StartAck, d.stdoutPipe)
	if err != nil {
		return "", errors.Trace(err)
	}

	// Send filenames to tar to remote
	_, err = d.stdinPipe.Write([]byte(filenames))
	if err != nil {
		return "", errors.Trace(err)
	}

	// Wait till remote wrote tar and sent us the tar size
	readString, err := readTill(EndAck, d.stderrPipe)
	if err != nil {
		return "", errors.Trace(err)
	}

	// Parse tar size
	splitted := strings.Split(readString, "\n")

	if splitted[len(splitted)-1] != EndAck {
		return "", fmt.Errorf("[Downstream] Cannot find %s in %s", EndAck, readString)
	}

	tarSize, err := strconv.ParseInt(splitted[len(splitted)-2], 10, 64)
	if err != nil {
		return "", errors.Trace(err)
	}
	if tarSize == 0 {
		return "", errors.New("[Downstream] Empty tar")
	}

	return d.downloadArchive(tarSize)
}

func (d *downstream) downloadArchive(tarSize int64) (string, error) {
	// Open file where tar will be written to
	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		return "", errors.Trace(err)
	}

	defer tempFile.Close()

	// Apply rate limit if specified
	var downloadReader io.Reader = d.stdoutPipe
	if d.config.DownstreamLimit > 0 {
		downloadReader = ratelimit.Reader(d.stdoutPipe, ratelimit.NewBucketWithRate(float64(d.config.DownstreamLimit), d.config.DownstreamLimit))
	}

	// Write From stdout to temp file
	bytesRead, err := io.CopyN(tempFile, downloadReader, tarSize)
	if err != nil {
		return "", errors.Trace(err)
	}
	if bytesRead != tarSize {
		return "", fmt.Errorf("[Downstream] Downloaded tar has wrong filesize: got %d, expected: %d", bytesRead, tarSize)
	}

	return tempFile.Name(), nil
}

func (d *downstream) removeFilesAndFolders(removeFiles map[string]*fileInformation) {
	d.config.fileIndex.fileMapMutex.Lock()
	defer d.config.fileIndex.fileMapMutex.Unlock()

	fileMap := d.config.fileIndex.fileMap

	// Remove Files & Folders
	numRemoveFiles := len(removeFiles)

	if numRemoveFiles > 3 {
		d.config.Logf("[Downstream] Remove %d files", numRemoveFiles)
	}

	for key, value := range removeFiles {
		absFilepath := filepath.Join(d.config.WatchPath, key)

		if shouldRemoveLocal(absFilepath, value, d.config) {
			if numRemoveFiles <= 3 || d.config.Verbose {
				d.config.Logf("[Downstream] Remove %s", key)
			}

			if value.IsDirectory {
				deleteSafeRecursive(d.config.WatchPath, key, fileMap, removeFiles, d.config)
			} else {
				err := os.Remove(absFilepath)
				if err != nil {
					if os.IsNotExist(err) == false {
						d.config.Logf("[Downstream] Skip file delete %s: %v", key, err)
					}
				}
			}
		}

		delete(fileMap, key)
	}
}

func (d *downstream) createFolders(createFolders []*fileInformation) {
	d.config.fileIndex.fileMapMutex.Lock()
	defer d.config.fileIndex.fileMapMutex.Unlock()

	fileMap := d.config.fileIndex.fileMap
	numCreateFolders := len(createFolders)

	// Create Folders
	if numCreateFolders > 3 {
		d.config.Logf("[Downstream] Create %d folders", len(createFolders))
	}

	for _, element := range createFolders {
		if element.IsDirectory {
			if numCreateFolders <= 3 || d.config.Verbose {
				d.config.Logln("[Downstream] Create folder: " + element.Name)
			}

			err := os.MkdirAll(path.Join(d.config.WatchPath, element.Name), 0755)
			if err != nil {
				d.config.Error(err)
			}

			if fileMap[element.Name] == nil {
				d.config.fileIndex.CreateDirInFileMap(element.Name)
			}
		}
	}
}
