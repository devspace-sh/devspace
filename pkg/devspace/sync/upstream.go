package sync

import (
	"archive/tar"
	"github.com/juju/errors"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/covexo/devspace/pkg/devspace/clients/kubectl"
	"github.com/rjeczalik/notify"
)

type upstream struct {
	events    chan notify.EventInfo
	interrupt chan bool
	config    *SyncConfig

	stdinPipe  io.WriteCloser
	stdoutPipe io.ReadCloser
	stderrPipe io.ReadCloser
}

func (u *upstream) start() error {
	u.events = make(chan notify.EventInfo, 10000) // High buffer size so we don't miss any fsevents if there are a lot of changes
	u.interrupt = make(chan bool, 1)

	err := u.startShell()

	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (u *upstream) diffServerClient(filepath string, sendChanges *[]*FileInformation, downloadChanges map[string]*FileInformation) error {
	relativePath := getRelativeFromFullPath(filepath, u.config.WatchPath)
	stat, err := os.Lstat(filepath)

	// We skip files that are suddenly not there anymore
	if err != nil {
		return nil
	}

	// We skip symlinks
	if stat.Mode()&os.ModeSymlink != 0 {
		return nil
	}

	// Exclude files on the exclude list
	if u.config.compExcludeRegEx != nil {
		for _, regExp := range u.config.compExcludeRegEx {
			if regExp.MatchString(relativePath) {
				return nil
			}
		}
	}

	delete(downloadChanges, relativePath)

	if stat.IsDir() {
		files, err := ioutil.ReadDir(filepath)

		if err != nil {
			u.config.Logf("[Upstream] Couldn't read dir %s: %s\n", filepath, err.Error())
			return nil
		}

		if len(files) == 0 {
			if u.config.fileMap[relativePath] == nil {
				*sendChanges = append(*sendChanges, &FileInformation{
					Name:        relativePath,
					IsDirectory: true,
				})
			}
		}

		for _, f := range files {
			if err := u.diffServerClient(path.Join(filepath, f.Name()), sendChanges, downloadChanges); err != nil {
				return errors.Trace(err)
			}
		}

		return nil
	} else {
		if u.config.fileMap[relativePath] == nil || ceilMtime(stat.ModTime()) > u.config.fileMap[relativePath].Mtime+1 {
			*sendChanges = append(*sendChanges, &FileInformation{
				Name:        relativePath,
				Mtime:       ceilMtime(stat.ModTime()),
				Size:        stat.Size(),
				IsDirectory: false,
			})
		}

		return nil
	}
}

func (u *upstream) collectChanges() error {
	for {
		var changes []*FileInformation

		changeAmount := 0

		for {
			select {
			case <-u.interrupt:
				return nil
			case event := <-u.events:
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

				changes = append(changes, u.getFileInformationFromEvent(events)...)
			case <-time.After(time.Millisecond * 600):
				break
			}

			// We gather changes till there are no more changes for 1 second
			if changeAmount == len(changes) && changeAmount > 0 {
				break
			}

			changeAmount = len(changes)
		}

		var files []*FileInformation

		lenChanges := len(changes)

		u.config.Logf("[Upstream] Processing %d changes\n", lenChanges)

		for index, element := range changes {
			if element.Mtime > 0 {
				if lenChanges <= 10 {
					u.config.Logf("[Upstream] Create %s\n", element.Name)
				}

				files = append(files, element)

				// Look ahead
				if len(changes) <= index+1 || changes[index+1].Mtime == 0 {
					err := u.sendFiles(files)

					if err != nil {
						return errors.Trace(err)
					}

					u.config.Logf("[Upstream] Successfully sent %d create changes\n", len(changes))

					files = make([]*FileInformation, 0, 10)
				}
			} else {
				if lenChanges <= 10 {
					u.config.Logf("[Upstream] Remove %s\n", element.Name)
				}

				files = append(files, element)

				// Look ahead
				if len(changes) <= index+1 || changes[index+1].Mtime > 0 {
					err := u.applyRemoves(files)

					if err != nil {
						return errors.Trace(err)
					}

					u.config.Logf("[Upstream] Successfully sent %d delete changes\n", len(changes))

					files = make([]*FileInformation, 0, 10)
				}
			}
		}
	}
}

func (u *upstream) getFileInformationFromEvent(events []notify.EventInfo) []*FileInformation {
	u.config.fileMapMutex.Lock()
	defer u.config.fileMapMutex.Unlock()

	changes := make([]*FileInformation, 0, len(events))

OUTER:
	for _, event := range events {
		fullpath := event.Path()
		relativePath := getRelativeFromFullPath(fullpath, u.config.WatchPath)

		if u.config.compExcludeRegEx != nil {
			for _, regExp := range u.config.compExcludeRegEx {
				if regExp.MatchString(relativePath) {
					continue OUTER // Path is excluded
				}
			}
		}

		stat, err := os.Stat(fullpath)

		if err == nil { // Does exist -> Create File or Folder
			if u.config.fileMap[relativePath] != nil {
				if stat.IsDir() {
					continue // Folder already exists
				} else {
					if ceilMtime(stat.ModTime()) == u.config.fileMap[relativePath].Mtime &&
						stat.Size() == u.config.fileMap[relativePath].Size {
						continue // File did not change or was changed by downstream
					}
				}
			}

			// New Create Task
			changes = append(changes, &FileInformation{
				Name:        relativePath,
				Mtime:       ceilMtime(stat.ModTime()),
				Size:        stat.Size(),
				IsDirectory: stat.IsDir(),
			})
		} else { // Does not exist -> Remove
			if u.config.fileMap[relativePath] == nil {
				continue // File / Folder was already deleted from map so event was already processed or should not be processed
			}

			// New Remove Task
			changes = append(changes, &FileInformation{
				Name: relativePath,
			})
		}
	}

	return changes
}

func (u *upstream) applyRemoves(files []*FileInformation) error {
	u.config.fileMapMutex.Lock()
	defer u.config.fileMapMutex.Unlock()

	u.config.Logf("[Upstream] Handling %d removes\n", len(files))

	// Send rm commands with max 50 input args
	for i := 0; i < len(files); i = i + 50 {
		rmCommand := "rm -R "
		removeArguments := 0

		for j := 0; j < 50 && i+j < len(files); j++ {
			relativePath := files[i+j].Name

			if u.config.fileMap[relativePath] != nil {
				relativePath = strings.Replace(relativePath, "'", "\\'", -1)
				rmCommand += "'" + u.config.DestPath + relativePath + "' "
				removeArguments++

				if u.config.fileMap[relativePath].IsDirectory {
					u.config.removeDirInFileMap(relativePath)
				} else {
					delete(u.config.fileMap, relativePath)
				}
			}
		}

		if removeArguments > 0 {
			rmCommand += " >/dev/null && printf \"" + EndAck + "\" || printf \"" + EndAck + "\"\n"
			// u.config.Logf("[Upstream] Handle command %s", rmCommand)

			if u.stdinPipe != nil {
				_, err := u.stdinPipe.Write([]byte(rmCommand))

				if err != nil {
					return errors.Trace(err)
				}

				waitTill(EndAck, u.stdoutPipe)
			}
		}
	}

	return nil
}

func (u *upstream) startShell() error {
	stdinPipe, stdoutPipe, stderrPipe, err := kubectl.Exec(u.config.Kubectl, u.config.Pod, u.config.Container.Name, []string{"sh"}, false)

	if err != nil {
		return errors.Trace(err)
	}

	u.stdinPipe = stdinPipe
	u.stdoutPipe = stdoutPipe
	u.stderrPipe = stderrPipe

	go func() {
		pipeStream(os.Stderr, u.stderrPipe)
	}()

	return nil
}

func (u *upstream) sendFiles(files []*FileInformation) error {
	filename, writtenFiles, err := u.writeTar(files)

	if err != nil {
		return errors.Trace(err)
	}

	if len(writtenFiles) == 0 {
		return nil
	}

	// u.config.Logf("[Upstream] Wrote changes to file %s\n", filename)

	f, err := os.Open(filename)

	if err != nil {
		return errors.Trace(err)
	}

	defer f.Close()
	stat, err := f.Stat()

	if err != nil {
		return errors.Trace(err)
	}

	if stat.Size()%512 != 0 {
		return errors.New("[Upstream] Tar archive has wrong size (Not dividable by 512)")
	}

	u.config.fileMapMutex.Lock()
	defer u.config.fileMapMutex.Unlock()

	// TODO: Implement timeout to prevent endless loop
	cmd := "fileSize=" + strconv.Itoa(int(stat.Size())) + `;
					tmpFile="/tmp/devspace-upstream";
					mkdir -p /tmp;
					mkdir -p '` + u.config.DestPath + `';

					pid=$$;
					cat </proc/$pid/fd/0 >"$tmpFile" &
					ddPid=$!;

					echo "` + StartAck + `";

					while true; do
							bytesRead=$(stat -c "%s" "$tmpFile" 2>/dev/null || printf "0");
						
							if [ "$bytesRead" = "$fileSize" ]; then
									kill $ddPid;
									break;
							fi;

							sleep 0.1;
					done;

					tar xf "$tmpFile" -C '` + u.config.DestPath + `/.' 2>/dev/null;
					echo "` + EndAck + `";
		` // We need that extra new line or otherwise the command is not sent

	if u.stdinPipe != nil {
		n, err := u.stdinPipe.Write([]byte(cmd))

		if err != nil {
			u.config.Logf("[Upstream] Writing to u.stdinPipe failed: %s\n", err.Error())

			return errors.Trace(err)
		}

		// Wait till confirmation
		err = waitTill(StartAck, u.stdoutPipe)

		if err != nil {
			return errors.Trace(err)
		}

		buf := make([]byte, 512, 512)

		for {
			n, err = f.Read(buf)

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

			n, err = u.stdinPipe.Write(buf)

			if err != nil {
				return errors.Trace(err)
			}

			if n < 512 {
				return errors.New("[Upstream] Only " + strconv.Itoa(n) + " Bytes written to stdin pipe (512 expected)")
			}
		}
	}

	// Delete file
	f.Close()
	err = os.Remove(f.Name())

	if err != nil {
		return errors.Trace(err)
	}

	// Wait till confirmation
	err = waitTill(EndAck, u.stdoutPipe)

	if err != nil {
		return errors.Trace(err)
	}

	// Update filemap
	for _, element := range writtenFiles {
		u.config.createDirInFileMap(path.Dir(element.Name))
		u.config.fileMap[element.Name] = element
	}

	return nil
}

func (u *upstream) writeTar(files []*FileInformation) (string, map[string]*FileInformation, error) {
	f, err := ioutil.TempFile("", "")

	if err != nil {
		return "", nil, errors.Trace(err)
	}

	defer f.Close()

	tarWriter := tar.NewWriter(f)
	defer tarWriter.Close()

	writtenFiles := make(map[string]*FileInformation)

	for _, element := range files {
		relativePath := element.Name

		if writtenFiles[relativePath] == nil {
			err := u.recursiveTar(u.config.WatchPath, relativePath, "", relativePath, writtenFiles, tarWriter)

			if err != nil {
				u.config.Logf("[Upstream] Tar failed: %s. Will retry in 4 seconds...\n", err.Error())
				os.Remove(f.Name())

				time.Sleep(time.Second * 4)

				return u.writeTar(files)
			}
		}
	}

	return f.Name(), writtenFiles, nil
}

// TODO: Error handling if files are not there
func (u *upstream) recursiveTar(srcBase, srcFile, destBase, destFile string, writtenFiles map[string]*FileInformation, tw *tar.Writer) error {
	filepath := path.Join(srcBase, srcFile)
	relativePath := getRelativeFromFullPath(filepath, srcBase)

	if writtenFiles[relativePath] != nil {
		return nil
	}

	stat, err := os.Lstat(filepath)

	// We skip files that are suddenly not there anymore
	if err != nil {
		u.config.Logf("[Upstream] Couldn't stat file %s: %s\n", filepath, err.Error())

		return nil
	}

	// We skip symlinks
	if stat.Mode()&os.ModeSymlink != 0 {
		return nil
	}

	fileInformation := &FileInformation{
		Name:        relativePath,
		Size:        stat.Size(),
		Mtime:       ceilMtime(stat.ModTime()),
		IsDirectory: stat.IsDir(),
	}

	if stat.IsDir() {
		files, err := ioutil.ReadDir(filepath)

		if err != nil {
			u.config.Logf("[Upstream] Couldn't read dir %s: %s\n", filepath, err.Error())
			return nil
		}

		if len(files) == 0 {
			//case empty directory
			hdr, _ := tar.FileInfoHeader(stat, filepath)
			hdr.Name = strings.Replace(destFile, "\\", "/", -1) // Need to replace \ with / for windows

			if err := tw.WriteHeader(hdr); err != nil {
				return errors.Trace(err)
			}

			writtenFiles[relativePath] = fileInformation
		}

		for _, f := range files {
			if err := u.recursiveTar(srcBase, path.Join(srcFile, f.Name()), destBase, path.Join(destFile, f.Name()), writtenFiles, tw); err != nil {
				return errors.Trace(err)
			}
		}

		return nil
	} else {
		f, err := os.Open(filepath)

		if err != nil {
			return errors.Trace(err)
		}

		defer f.Close()

		//case regular file or other file type like pipe
		hdr, err := tar.FileInfoHeader(stat, filepath)

		if err != nil {
			return errors.Trace(err)
		}

		hdr.Name = strings.Replace(destFile, "\\", "/", -1)

		if err := tw.WriteHeader(hdr); err != nil {
			return errors.Trace(err)
		}

		if _, err := io.Copy(tw, f); err != nil {
			return errors.Trace(err)
		}

		writtenFiles[relativePath] = fileInformation

		return f.Close()
	}
}
