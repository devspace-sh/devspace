package sync

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/juju/errors"

	"github.com/covexo/devspace/pkg/devspace/clients/kubectl"
)

const IsDirectory uint64 = 040000
const IsRegularFile uint64 = 0100000
const IsSymbolicLink uint64 = 0120000

type downstream struct {
	interrupt chan bool
	config    *SyncConfig

	stdinPipe  io.WriteCloser
	stdoutPipe io.ReadCloser
	stderrPipe io.ReadCloser
}

type parsingError struct {
	msg string
}

func (p parsingError) Error() string {
	return p.msg
}

func (d *downstream) start() error {
	d.interrupt = make(chan bool, 1)

	err := d.startShell()

	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (d *downstream) populateFileMap() error {
	createFiles := make([]*fileInformation, 0, 128)

	cmd := "mkdir -p '" + d.config.DestPath + "' && find '" + d.config.DestPath + "' -exec stat -c \"%n///%s,%Y,%f\" {} + 2>/dev/null && echo -n \"" + EndAck + "\" || echo \"" + ErrorAck + "\"\n"
	_, err := d.stdinPipe.Write([]byte(cmd))

	if err != nil {
		return errors.Trace(err)
	}

	err = d.collectChanges(&createFiles, nil)

	if err != nil {
		if _, ok := err.(parsingError); ok {
			time.Sleep(time.Second * 4)

			return d.populateFileMap()
		}

		return errors.Trace(err)
	}

	d.config.fileIndex.ExecuteSafe(func(fileMap map[string]*fileInformation) {
		for _, element := range createFiles {
			fileMap[element.Name] = element
		}
	})

	return nil
}

func (d *downstream) startShell() error {
	stdinPipe, stdoutPipe, stderrPipe, err := kubectl.Exec(d.config.Kubectl, d.config.Pod, d.config.Container.Name, []string{"sh"}, false)

	if err != nil {
		return errors.Trace(err)
	}

	d.stdinPipe = stdinPipe
	d.stdoutPipe = stdoutPipe
	d.stderrPipe = stderrPipe

	return nil
}

func (d *downstream) mainLoop() error {
	lastAmountChanges := 0

	for {
		createFiles := make([]*fileInformation, 0, 128)
		removeFiles := make(map[string]*fileInformation)

		d.config.fileIndex.ExecuteSafe(func(fileMap map[string]*fileInformation) {
			for key, value := range fileMap {
				removeFiles[key] = &fileInformation{
					Name:        value.Name,
					Size:        value.Size,
					Mtime:       value.Mtime,
					IsDirectory: value.IsDirectory,
				}
			}
		})

		cmd := "mkdir -p '" + d.config.DestPath + "' && find '" + d.config.DestPath + "' -exec stat -c \"%n///%s,%Y,%f\" {} + 2>/dev/null && echo -n \"" + EndAck + "\" || echo \"" + ErrorAck + "\"\n"
		_, err := d.stdinPipe.Write([]byte(cmd))

		if err != nil {
			return errors.Trace(err)
		}

		err = d.collectChanges(&createFiles, removeFiles)

		if err != nil {
			if _, ok := err.(parsingError); ok {
				time.Sleep(time.Second * 4)
				continue
			}

			return errors.Trace(err)
		}

		amountChanges := len(createFiles) + len(removeFiles)

		if amountChanges > 0 {
			d.config.Logf("[Downstream] Collected %d changes\n", amountChanges)
		}

		if lastAmountChanges > 0 && amountChanges == lastAmountChanges {
			err = d.applyChanges(createFiles, removeFiles)

			if err != nil {
				return errors.Trace(err)
			}
		}

		select {
		case <-d.interrupt:
			return nil
		case <-time.After(2 * time.Second):
			break
		}

		lastAmountChanges = len(createFiles) + len(removeFiles)
	}
}

func (d *downstream) applyChanges(createFiles []*fileInformation, removeFiles map[string]*fileInformation) error {
	var err error
	downloadFiles := make([]*fileInformation, 0, int(len(createFiles)/2))
	createFolders := make([]*fileInformation, 0, int(len(createFiles)/2))
	tempDownloadpath := ""

	// Determine folder creates and file creates and seperate them
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

	d.config.fileIndex.ExecuteSafe(func(fileMap map[string]*fileInformation) {
		d.removeFilesAndFolders(fileMap, removeFiles)
	})

	d.config.fileIndex.ExecuteSafe(func(fileMap map[string]*fileInformation) {
		d.createFolders(fileMap, createFolders)
	})

	if len(downloadFiles) > 0 {
		f, err := os.Open(tempDownloadpath)

		if err != nil {
			return errors.Trace(err)
		}

		defer f.Close()

		d.config.Logln("[Downstream] Start untaring")
		err = untarAll(f, d.config.WatchPath, d.config.DestPath, d.config) // This can be a lengthy process
		d.config.Logln("[Downstream] End untaring")

		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (d *downstream) downloadFiles(files []*fileInformation) (string, error) {
	var buffer bytes.Buffer

	lenFiles := len(files)

	if lenFiles > 10 {
		d.config.Logf("[Downstream] Download %d files\n", lenFiles)
	}

	// Download Files
	for _, element := range files {
		if lenFiles <= 10 {
			d.config.Logln("[Downstream] Download file - " + element.Name)
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

					echo "START";

					while true; do
							bytesRead=$(stat -c "%s" "$tmpFileInput" 2>/dev/null || printf "0");
						
							if [ "$bytesRead" = "$fileSize" ]; then
									kill $ddPid;
									break;
							fi;

							sleep 0.1;
					done;
					tar -cf "$tmpFileOutput" -T "$tmpFileInput" 2>/dev/null;
					(>&2 echo "` + StartAck + `");
					(>&2 echo $(stat -c "%s" "$tmpFileOutput"));
					(>&2 echo "` + EndAck + `");
					cat "$tmpFileOutput";
		` // We need that extra new line, otherwise the command is not executed properly

	tempFile, err := ioutil.TempFile("", "")
	defer tempFile.Close()

	if err != nil {
		return "", errors.Trace(err)
	}

	d.stdinPipe.Write([]byte(cmd))
	err = waitTill(StartAck, d.stdoutPipe)

	if err != nil {
		return "", errors.Trace(err)
	}

	d.stdinPipe.Write([]byte(filenames))
	readString, err := readTill(EndAck, d.stderrPipe)

	if err != nil {
		return "", errors.Trace(err)
	}

	tarSize := 0
	splitted := strings.Split(readString, "\n")

	if splitted[len(splitted)-1] == EndAck {
		tarSize, err = strconv.Atoi(splitted[len(splitted)-2])

		if err != nil {
			return "", errors.Trace(err)
		}
	} else {
		return "", errors.New("[Downstream] Cannot find DONE in " + readString)
	}

	if tarSize == 0 || tarSize%512 != 0 {
		return "", errors.New("[Downstream] Invalid tarSize: " + strconv.Itoa(tarSize))
	}

	d.config.Logln("[Downstream] Write tar with " + strconv.Itoa(tarSize) + " size")

	// Write From stdout
	buf := make([]byte, 512, 512)

	for bytesLeft := tarSize; bytesLeft > 0; {
		n, err := d.stdoutPipe.Read(buf)

		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}

			return "", errors.Trace(err)
		}

		// process buf
		if err != nil && err != io.EOF {
			return "", errors.Trace(err)
		}

		n, err = tempFile.Write(buf)

		if err != nil {
			return "", errors.Trace(err)
		}

		// d.config.Logln("Wrote " + strconv.Itoa(n) + " bytes")
		bytesLeft -= n
	}

	d.config.Logln("[Downstream] Wrote Tempfile " + tempFile.Name())

	return tempFile.Name(), nil
}

func (d *downstream) removeFilesAndFolders(fileMap map[string]*fileInformation, removeFiles map[string]*fileInformation) {
	// Remove Files & Folders
	numRemoveFiles := len(removeFiles)

	if numRemoveFiles > 10 {
		d.config.Logf("[Downstream] Remove %d files\n", numRemoveFiles)
	}

	// A file is only deleted if the following conditions are met:
	// - The file name is present in the d.config.fileMap map
	// - The file did not change in terms of size and mtime in the d.config.fileMap since we started the collecting changes process
	// - The file is present on the filesystem and did not change in terms of size and mtime on the filesystem
	for key, value := range removeFiles {
		if value != nil && fileMap[key] != nil {
			if numRemoveFiles <= 10 {
				d.config.Logf("[Downstream] Remove %s\n", key)
			}

			if fileMap[key].IsDirectory {
				deleteSafeRecursive(d.config.WatchPath, key, fileMap, removeFiles, d.config)
			} else {
				if value.Mtime == fileMap[key].Mtime && value.Size == fileMap[key].Size {
					if deleteSafe(path.Join(d.config.WatchPath, key), fileMap[key]) == false {
						d.config.Logf("[Downstream] Skip file delete %s\n", key)
					}
				}

				delete(fileMap, key)
			}
		}
	}
}

func (d *downstream) createFolders(fileMap map[string]*fileInformation, createFolders []*fileInformation) {
	numCreateFolders := len(createFolders)

	// Create Folders
	if numCreateFolders > 10 {
		d.config.Logf("[Downstream] Create %d folders\n", len(createFolders))
	}

	for _, element := range createFolders {
		if fileMap[element.Name] == nil && element.IsDirectory {
			if numCreateFolders <= 10 {
				d.config.Logln("[Downstream] Create folder: " + element.Name)
			}

			err := os.MkdirAll(path.Join(d.config.WatchPath, element.Name), 0755)

			if err != nil {
				d.config.Logln(err)
			}

			d.config.fileIndex.CreateDirInFileMap(element.Name)
		}
	}
}

func (d *downstream) collectChanges(createFiles *[]*fileInformation, removeFiles map[string]*fileInformation) error {
	buf := make([]byte, 0, 512)
	overlap := ""

	for {
		n, err := d.stdoutPipe.Read(buf[:cap(buf)])

		buf = buf[:n]

		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}

			return errors.Trace(err)
		}

		// process buf
		if err != nil && err != io.EOF {
			return errors.Trace(err)
		}

		lines := strings.Split(string(buf), "\n")

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
				return nil
			} else if line == ErrorAck || overlap == ErrorAck {
				return parsingError{
					msg: "Parsing Error",
				}
			} else if line != "" {
				err = d.evaluateFile(line, createFiles, removeFiles)

				if err != nil {
					return errors.Trace(err)
				}
			}
		}
	}

	return nil
}

func (d *downstream) evaluateFile(fileline string, createFiles *[]*fileInformation, removeFiles map[string]*fileInformation) error {
	d.config.fileIndex.fileMapMutex.Lock()
	defer d.config.fileIndex.fileMapMutex.Unlock()

	fileMap := d.config.fileIndex.fileMap
	fileInformation, err := parseFileInformation(fileline, d.config.DestPath, d.config.ignoreMatcher)

	if err != nil {
		return errors.Trace(err)
	}

	if fileInformation == nil {
		return nil
	}

	if removeFiles[fileInformation.Name] != nil {
		delete(removeFiles, fileInformation.Name)
	}

	if fileMap[fileInformation.Name] != nil {
		if fileInformation.IsDirectory == false {
			if (fileInformation.Mtime >= fileMap[fileInformation.Name].Mtime && fileInformation.Size != fileMap[fileInformation.Name].Size) ||
				fileInformation.Mtime > fileMap[fileInformation.Name].Mtime {
				*createFiles = append(*createFiles, fileInformation)
			}
		}
	} else {
		*createFiles = append(*createFiles, fileInformation)
	}

	return nil
}
