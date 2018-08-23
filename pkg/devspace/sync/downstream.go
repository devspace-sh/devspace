package sync

import (
	"archive/tar"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
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

type ParsingError struct {
	msg string
}

func (p ParsingError) Error() string {
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

func (d *downstream) mainLoop() error {
	lastAmountChanges := 0

	for {
		createFiles := make([]*FileInformation, 0, 128)
		removeFiles := make(map[string]*FileInformation)

		d.config.fileMapMutex.Lock()

		for key, value := range d.config.fileMap {
			removeFiles[key] = &FileInformation{
				Name:        value.Name,
				Size:        value.Size,
				Mtime:       value.Mtime,
				IsDirectory: value.IsDirectory,
			}
		}

		d.config.fileMapMutex.Unlock()

		cmd := "mkdir -p '" + d.config.DestPath + "' && find '" + d.config.DestPath + "' -exec stat -c \"%n///%s,%Y,%f\" {} + 2>/dev/null && echo -n \"" + EndAck + "\" || echo \"" + ErrorAck + "\"\n"
		_, err := d.stdinPipe.Write([]byte(cmd))

		if err != nil {
			return errors.Trace(err)
		}

		err = d.collectChanges(&createFiles, removeFiles)

		if err != nil {
			if _, ok := err.(ParsingError); ok {
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

func (d *downstream) populateFileMap() error {
	createFiles := make([]*FileInformation, 0, 128)

	cmd := "mkdir -p '" + d.config.DestPath + "' && find '" + d.config.DestPath + "' -exec stat -c \"%n///%s,%Y,%f\" {} + 2>/dev/null && echo -n \"" + EndAck + "\" || echo \"" + ErrorAck + "\"\n"
	_, err := d.stdinPipe.Write([]byte(cmd))

	if err != nil {
		return errors.Trace(err)
	}

	err = d.collectChanges(&createFiles, nil)

	if err != nil {
		if _, ok := err.(ParsingError); ok {
			time.Sleep(time.Second * 4)

			return d.populateFileMap()
		}

		return errors.Trace(err)
	}

	d.config.fileMapMutex.Lock()
	defer d.config.fileMapMutex.Unlock()

	for _, element := range createFiles {
		d.config.fileMap[element.Name] = element
	}

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

	// go func() {
	//	pipeStream(os.Stderr, d.stderrPipe)
	// }()

	return nil
}

func (d *downstream) applyChanges(createFiles []*FileInformation, removeFiles map[string]*FileInformation) error {
	var err error
	downloadFiles := make([]*FileInformation, 0, int(len(createFiles)/2))
	createFolders := make([]*FileInformation, 0, int(len(createFiles)/2))
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

	d.config.fileMapMutex.Lock()

	// Remove Files & Folders
	numRemoveFiles := len(removeFiles)

	if numRemoveFiles > 10 {
		d.config.Logf("[Downstream] Remove %d files\n", numRemoveFiles)
	}

	for key, value := range removeFiles {
		if value != nil && d.config.fileMap[key] != nil {
			if numRemoveFiles <= 10 {
				d.config.Logf("[Downstream] Remove %s\n", key)
			}

			if d.config.fileMap[key].IsDirectory {
				d.deleteSafeRecursive(d.config.WatchPath, key)
			} else {
				if deleteSafe(path.Join(d.config.WatchPath, key), d.config.fileMap[key]) == false {
					d.config.Logf("[Downstream] Skip file delete %s\n", key)
				}
			}

			delete(d.config.fileMap, key)
		}
	}

	numCreateFolders := len(createFolders)

	// Create Folders
	if numCreateFolders > 10 {
		d.config.Logf("[Downstream] Create %d folders\n", len(createFolders))
	}

	for _, element := range createFolders {
		if d.config.fileMap[element.Name] == nil && element.IsDirectory {
			if numCreateFolders <= 10 {
				d.config.Logln("[Downstream] Create folder: " + element.Name)
			}

			err := os.MkdirAll(path.Join(d.config.WatchPath, element.Name), 0755)

			if err != nil {
				d.config.Logln(err)
			}

			d.config.createDirInFileMap(element.Name)
		}
	}

	d.config.fileMapMutex.Unlock()

	if len(downloadFiles) > 0 {
		f, err := os.Open(tempDownloadpath)

		if err != nil {
			return errors.Trace(err)
		}

		defer f.Close()

		d.config.Logln("[Downstream] Start untaring")
		err = d.untarAll(f, d.config.WatchPath, d.config.DestPath) // This can be a lengthy process
		d.config.Logln("[Downstream] End untaring")

		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (d *downstream) deleteSafeRecursive(basepath string, relativePath string) {
	absolutePath := path.Join(basepath, relativePath)
	relativePath = getRelativeFromFullPath(absolutePath, basepath)

	// We don't delete the folder or the contents if we haven't tracked it
	if d.config.fileMap[relativePath] == nil {
		d.config.Logf("[Downstream] Skip delete directory %s\n", relativePath)

		return
	}

	files, err := ioutil.ReadDir(absolutePath)

	if err != nil {
		return
	}

	for _, f := range files {
		if f.IsDir() {
			d.deleteSafeRecursive(basepath, path.Join(relativePath, f.Name()))
		} else {
			filepath := path.Join(relativePath, f.Name())

			// We don't delete the file if we haven't tracked it
			if d.config.fileMap[filepath] != nil {
				if deleteSafe(path.Join(basepath, filepath), d.config.fileMap[filepath]) == false {
					d.config.Logf("[Downstream] Skip file delete %s\n", relativePath)
				}
			} else {
				d.config.Logf("[Downstream] Skip file delete %s\n", relativePath)
			}
		}
	}

	if d.config.fileMap[relativePath] != nil {
		os.Remove(absolutePath) // This will not remove the directory if there is still a file or directory in it
	}
}

func (d *downstream) untarNext(tarReader *tar.Reader, entrySeq int, destPath, prefix string) (bool, error) {
	d.config.fileMapMutex.Lock()
	defer d.config.fileMapMutex.Unlock()

	header, err := tarReader.Next()
	if err != nil {
		if err != io.EOF {
			return false, errors.Trace(err)
		}

		return false, nil
	}

	mode := header.FileInfo().Mode()
	relativePath := clean(header.Name[len(prefix):])
	outFileName := path.Join(destPath, relativePath)
	baseName := path.Dir(outFileName)

	// Check if newer file is there and then don't override?
	stat, err := os.Stat(outFileName)

	if err == nil {
		if ceilMtime(stat.ModTime()) > header.FileInfo().ModTime().Unix() {
			d.config.Logf("[Downstream] Don't override %s because file has newer mTime timestamp\n", relativePath)
			return true, nil
		}
	}

	if err := os.MkdirAll(baseName, 0755); err != nil {
		return false, errors.Trace(err)
	}

	if header.FileInfo().IsDir() {
		if err := os.MkdirAll(outFileName, 0755); err != nil {
			return false, errors.Trace(err)
		}

		d.config.createDirInFileMap(relativePath)

		return true, nil
	} else {
		d.config.createDirInFileMap(getRelativeFromFullPath(baseName, destPath))
	}

	// handle coping remote file into local directory
	if entrySeq == 0 && !header.FileInfo().IsDir() {
		exists, err := dirExists(outFileName)
		if err != nil {
			return false, errors.Trace(err)
		}
		if exists {
			outFileName = filepath.Join(outFileName, path.Base(clean(header.Name)))
		}
	}

	if mode&os.ModeSymlink != 0 {
		// Skip the processing of symlinks for now, because windows has problems with them
		// err := os.Symlink(header.Linkname, outFileName)
		// if err != nil {
		//	 return errors.Trace(err)
		// }
	} else {
		outFile, err := os.Create(outFileName)

		if err != nil {
			// Try again after 5 seconds
			time.Sleep(time.Second * 5)
			outFile, err = os.Create(outFileName)

			if err != nil {
				return false, errors.Trace(err)
			}
		}

		defer outFile.Close()

		if _, err := io.Copy(outFile, tarReader); err != nil {
			return false, errors.Trace(err)
		}

		if err := outFile.Close(); err != nil {
			return false, errors.Trace(err)
		}

		err = os.Chtimes(outFileName, time.Now(), header.FileInfo().ModTime())

		if err != nil {
			return false, errors.Trace(err)
		}

		relativePath = getRelativeFromFullPath(outFileName, destPath)

		// Update fileMap so that upstream does not upload the file
		d.config.fileMap[relativePath] = &FileInformation{
			Name:        relativePath,
			Mtime:       header.FileInfo().ModTime().Unix(),
			Size:        header.FileInfo().Size(),
			IsDirectory: false,
		}
	}

	return true, nil
}

func (d *downstream) untarAll(reader io.Reader, destPath, prefix string) error {
	entrySeq := 0

	// TODO: use compression here?
	tarReader := tar.NewReader(reader)

	for {
		shouldContinue, err := d.untarNext(tarReader, entrySeq, destPath, prefix)

		if err != nil {
			return errors.Trace(err)
		} else if shouldContinue == false {
			return nil
		}

		entrySeq++

		if entrySeq%500 == 0 {
			d.config.Logf("[Downstream] Untared %d files...\n", entrySeq)
		}
	}
}

func (d *downstream) downloadFiles(files []*FileInformation) (string, error) {
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

func (d *downstream) collectChanges(createFiles *[]*FileInformation, removeFiles map[string]*FileInformation) error {
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
				return ParsingError{
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

func (d *downstream) evaluateFile(fileline string, createFiles *[]*FileInformation, removeFiles map[string]*FileInformation) error {
	d.config.fileMapMutex.Lock()
	defer d.config.fileMapMutex.Unlock()

	fileinformation, err := d.parseFileInformation(fileline)

	if err != nil {
		return errors.Trace(err)
	}

	if fileinformation == nil {
		return nil
	}

	if removeFiles[fileinformation.Name] != nil {
		delete(removeFiles, fileinformation.Name)
	}

	if d.config.fileMap[fileinformation.Name] != nil {
		if fileinformation.IsDirectory == false {
			if (fileinformation.Mtime >= d.config.fileMap[fileinformation.Name].Mtime && fileinformation.Size != d.config.fileMap[fileinformation.Name].Size) ||
				fileinformation.Mtime > d.config.fileMap[fileinformation.Name].Mtime {
				*createFiles = append(*createFiles, fileinformation)
			}
		}
	} else {
		*createFiles = append(*createFiles, fileinformation)
	}

	return nil
}

func (d *downstream) parseFileInformation(fileline string) (*FileInformation, error) {
	fileinfo := FileInformation{}

	t := strings.Split(fileline, "///")

	if len(t) != 2 {
		return nil, errors.New("[Downstream] Wrong fileline: " + fileline)
	}

	if len(t[0]) <= len(d.config.DestPath) {
		return nil, nil
	}

	fileinfo.Name = t[0][len(d.config.DestPath):]

	if d.config.ignoreMatcher != nil {
		if d.config.ignoreMatcher.MatchesPath(fileinfo.Name) {
			return nil, nil
		}
	}

	t = strings.Split(t[1], ",")

	if len(t) != 3 {
		return nil, errors.New("[Downstream] Wrong fileline: " + fileline)
	}

	size, err := strconv.Atoi(t[0])

	if err != nil {
		return nil, errors.Trace(err)
	}

	fileinfo.Size = int64(size)

	mTime, err := strconv.Atoi(t[1])

	if err != nil {
		return nil, errors.Trace(err)
	}

	fileinfo.Mtime = int64(mTime)

	rawMode, err := strconv.ParseUint(t[2], 16, 32) // Parse hex string into uint64

	if err != nil {
		return nil, errors.Trace(err)
	}

	// We skip symbolic links for now, because windows has problems with them
	if rawMode&IsSymbolicLink == IsSymbolicLink {
		return nil, nil
	}

	fileinfo.IsDirectory = (rawMode & IsDirectory) == IsDirectory

	return &fileinfo, nil
}
