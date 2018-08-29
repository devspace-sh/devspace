package sync

import (
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/juju/errors"

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

func (u *upstream) collectChanges() error {
	for {
		var changes []*fileInformation

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

				u.config.fileIndex.ExecuteSafe(func(fileMap map[string]*fileInformation) {
					changes = append(changes, u.getfileInformationFromEvent(fileMap, events)...)
				})
			case <-time.After(time.Millisecond * 600):
				break
			}

			// We gather changes till there are no more changes for 1 second
			if changeAmount == len(changes) && changeAmount > 0 {
				break
			}

			changeAmount = len(changes)
		}

		var files []*fileInformation

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

					files = make([]*fileInformation, 0, 10)
				}
			} else {
				if lenChanges <= 10 {
					u.config.Logf("[Upstream] Remove %s\n", element.Name)
				}

				files = append(files, element)

				// Look ahead
				if len(changes) <= index+1 || changes[index+1].Mtime > 0 {
					err := u.config.fileIndex.ExecuteSafeError(func(fileMap map[string]*fileInformation) error {
						return u.applyRemoves(fileMap, files)
					})

					if err != nil {
						return errors.Trace(err)
					}

					u.config.Logf("[Upstream] Successfully sent %d delete changes\n", len(changes))

					files = make([]*fileInformation, 0, 10)
				}
			}
		}
	}
}

func (u *upstream) getfileInformationFromEvent(fileMap map[string]*fileInformation, events []notify.EventInfo) []*fileInformation {
	changes := make([]*fileInformation, 0, len(events))

OUTER:
	for _, event := range events {
		fullpath := event.Path()
		relativePath := getRelativeFromFullPath(fullpath, u.config.WatchPath)

		// Exclude files on the exclude list
		if u.config.ignoreMatcher != nil {
			if u.config.ignoreMatcher.MatchesPath(relativePath) {
				continue OUTER // Path is excluded
			}
		}

		stat, err := os.Stat(fullpath)

		if err == nil { // Does exist -> Create File or Folder
			if fileMap[relativePath] != nil {
				if stat.IsDir() {
					continue // Folder already exists
				} else {
					if ceilMtime(stat.ModTime()) == fileMap[relativePath].Mtime &&
						stat.Size() == fileMap[relativePath].Size {
						continue // File did not change or was changed by downstream
					}
				}
			}

			// New Create Task
			changes = append(changes, &fileInformation{
				Name:        relativePath,
				Mtime:       ceilMtime(stat.ModTime()),
				Size:        stat.Size(),
				IsDirectory: stat.IsDir(),
			})
		} else { // Does not exist -> Remove
			if fileMap[relativePath] == nil {
				continue // File / Folder was already deleted from map so event was already processed or should not be processed
			}

			// New Remove Task
			changes = append(changes, &fileInformation{
				Name: relativePath,
			})
		}
	}

	return changes
}

func (u *upstream) applyRemoves(fileMap map[string]*fileInformation, files []*fileInformation) error {
	u.config.Logf("[Upstream] Handling %d removes\n", len(files))

	// Send rm commands with max 50 input args
	for i := 0; i < len(files); i = i + 50 {
		rmCommand := "rm -R "
		removeArguments := 0

		for j := 0; j < 50 && i+j < len(files); j++ {
			relativePath := files[i+j].Name

			if fileMap[relativePath] != nil {
				relativePath = strings.Replace(relativePath, "'", "\\'", -1)
				rmCommand += "'" + u.config.DestPath + relativePath + "' "
				removeArguments++

				if fileMap[relativePath].IsDirectory {
					u.config.fileIndex.RemoveDirInFileMap(relativePath)
				} else {
					delete(fileMap, relativePath)
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
	stdinPipe, stdoutPipe, stderrPipe, err := kubectl.Exec(u.config.Kubectl, u.config.Pod, u.config.Container.Name, []string{"sh"}, false, nil)

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

func (u *upstream) sendFiles(files []*fileInformation) error {
	filename, writtenFiles, err := writeTar(files, u.config)

	if err != nil {
		return errors.Trace(err)
	}

	if len(writtenFiles) == 0 {
		return nil
	}

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

	return u.upload(f, strconv.Itoa(int(stat.Size())), writtenFiles)
}

func (u *upstream) upload(file *os.File, fileSize string, writtenFiles map[string]*fileInformation) error {
	u.config.fileIndex.fileMapMutex.Lock()
	defer u.config.fileIndex.fileMapMutex.Unlock()

	// TODO: Implement timeout to prevent endless loop
	cmd := "fileSize=" + fileSize + `;
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
		_, err := u.stdinPipe.Write([]byte(cmd))

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
			n, err := file.Read(buf)

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
	file.Close()
	err := os.Remove(file.Name())

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
		u.config.fileIndex.CreateDirInFileMap(path.Dir(element.Name))
		u.config.fileIndex.fileMap[element.Name] = element
	}

	return nil
}
