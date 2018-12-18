package sync

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/juju/ratelimit"

	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/rjeczalik/notify"
)

type upstream struct {
	events    chan notify.EventInfo
	interrupt chan bool
	config    *SyncConfig

	symlinks map[string]*Symlink

	stdinPipe  io.WriteCloser
	stdoutPipe io.ReadCloser
	stderrPipe io.ReadCloser
}

func (u *upstream) start() error {
	u.events = make(chan notify.EventInfo, 5000) // High buffer size so we don't miss any fsevents if there are a lot of changes
	u.symlinks = make(map[string]*Symlink)
	u.interrupt = make(chan bool, 1)

	err := u.startShell()

	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (u *upstream) startShell() error {
	if u.config.testing == false {
		stdinReader, stdinWriter, _ := os.Pipe()
		stdoutReader, stdoutWriter, _ := os.Pipe()
		stderrReader, stderrWriter, _ := os.Pipe()

		go func() {
			err := kubectl.ExecStream(u.config.Kubectl, u.config.Pod, u.config.Container.Name, []string{"sh"}, false, stdinReader, stdoutWriter, stderrWriter)
			if err != nil {
				u.config.Error(err)
			}
		}()

		u.stdinPipe = stdinWriter
		u.stdoutPipe = stdoutReader
		u.stderrPipe = stderrReader

		//go func() {
		//	pipeStream(os.Stderr, u.stderrPipe)
		//}()
	} else {
		var err error

		cmd := exec.Command("sh")

		u.stdinPipe, err = cmd.StdinPipe()
		if err != nil {
			return err
		}

		u.stdoutPipe, err = cmd.StdoutPipe()
		if err != nil {
			return err
		}

		u.stderrPipe, err = cmd.StderrPipe()
		if err != nil {
			return err
		}

		err = cmd.Start()
		if err != nil {
			return err
		}

		go func() {
			pipeStream(os.Stderr, u.stderrPipe)
		}()
	}

	return nil
}

func (u *upstream) mainLoop() error {
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

				fileInformations, err := u.getfileInformationFromEvent(events)
				if err != nil {
					return errors.Trace(err)
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
			return errors.Trace(err)
		}
	}
}

func (u *upstream) getfileInformationFromEvent(events []notify.EventInfo) ([]*fileInformation, error) {
	u.config.fileIndex.fileMapMutex.Lock()
	defer u.config.fileIndex.fileMapMutex.Unlock()

	fileMap := u.config.fileIndex.fileMap
	changes := make([]*fileInformation, 0, len(events))

	for _, event := range events {
		fileInfo, ok := event.(*fileInformation)

		if ok {
			changes = append(changes, fileInfo)
		} else {
			fullpath := event.Path()
			relativePath := getRelativeFromFullPath(fullpath, u.config.WatchPath)

			// Determine what kind of change we got (Create or Remove)
			newChange, err := evaluateChange(u.config, fileMap, relativePath, fullpath)
			if err != nil {
				return nil, errors.Trace(err)
			}

			if newChange != nil {
				changes = append(changes, newChange)
			}
		}
	}

	return changes, nil
}

func evaluateChange(s *SyncConfig, fileMap map[string]*fileInformation, relativePath, fullpath string) (*fileInformation, error) {
	stat, err := os.Lstat(fullpath)

	// File / Folder exist -> Create File or Folder
	// if File / Folder does not exist, we create a new remove change
	if err == nil {
		// Is symbolic link
		if stat.Mode()&os.ModeSymlink != 0 {
			_, symlinkExists := s.upstream.symlinks[fullpath]

			// Add symlink to map
			stat, err = s.upstream.AddSymlink(fullpath)
			if err != nil {
				return nil, errors.Trace(err)
			}
			if stat == nil {
				return nil, nil
			}

			// Only crawl if symlink wasn't there before and it is a directory
			if symlinkExists == false && stat.IsDir() {
				// Crawl all linked files & folders
				err = s.upstream.symlinks[fullpath].Crawl()
				if err != nil {
					return nil, errors.Trace(err)
				}
			}

			log.Infof("Symlink created/changed at %s", fullpath)
		}

		// Exclude changes on the upload exclude list
		if s.uploadIgnoreMatcher != nil {
			if s.uploadIgnoreMatcher.MatchesPath(relativePath) {
				// Add to file map and prevent download if local file is newer than the remote one
				if s.fileIndex.fileMap[relativePath] != nil && s.fileIndex.fileMap[relativePath].Mtime < roundMtime(stat.ModTime()) {
					// Add it to the fileMap
					s.fileIndex.fileMap[relativePath] = &fileInformation{
						Name:        relativePath,
						Mtime:       roundMtime(stat.ModTime()),
						Size:        stat.Size(),
						IsDirectory: stat.IsDir(),
					}
				}

				return nil, nil
			}
		}

		if shouldUpload(relativePath, stat, s, false) {
			// New Create Task
			return &fileInformation{
				Name:        relativePath,
				Mtime:       roundMtime(stat.ModTime()),
				Size:        stat.Size(),
				IsDirectory: stat.IsDir(),
			}, nil
		}
	} else {
		// Remove symlinks
		s.upstream.RemoveSymlinks(fullpath)

		// Check if we should remove path remote
		if shouldRemoveRemote(relativePath, s) {
			// New Remove Task
			return &fileInformation{
				Name: relativePath,
			}, nil
		}
	}

	return nil, nil
}

func (u *upstream) AddSymlink(absPath string) (os.FileInfo, error) {
	// Get real path
	targetPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		u.config.Logf("Warning: resolving symlink of %s: %v", absPath, err)
		return nil, nil // fmt.Errorf("Error resolving symlink of %s: %v", absPath, err)
	}

	stat, err := os.Lstat(targetPath)
	if err != nil {
		u.config.Logf("Warning: stating symlink %s: %v", targetPath, err)
		return nil, nil // fmt.Errorf("Error stating symlink %s: %v", targetPath, err)
	}

	// Check if we run into a recursive symlink
	if _, ok := u.symlinks[absPath]; ok {
		return stat, nil
	}

	symlink, err := NewSymlink(u, absPath, targetPath, stat.IsDir())
	if err != nil {
		return nil, fmt.Errorf("Cannot create symlink object for %s: %v", absPath, err)
	}

	u.symlinks[absPath] = symlink

	return stat, nil
}

func (u *upstream) RemoveSymlinks(absPath string) {
	for key, symlink := range u.symlinks {
		if key == absPath || strings.Index(strings.Replace(key, "\\", "/", -1)+"/", strings.Replace(absPath, "\\", "/", -1)) == 0 {
			log.Infof("Symlink deleted at %s", key)

			symlink.Stop()
			delete(u.symlinks, key)
		}
	}
}

func (u *upstream) applyChanges(changes []*fileInformation) error {
	var creates []*fileInformation
	var removes []*fileInformation

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
	err := u.applyRemoves(removes)
	if err != nil {
		return errors.Trace(err)
	}

	// Apply creates
	err = u.applyCreates(creates)
	if err != nil {
		return errors.Trace(err)
	}

	u.config.Logf("[Upstream] Successfully processed %d change(s)", len(changes))
	return nil
}

func (u *upstream) applyCreates(files []*fileInformation) error {
	filename, writtenFiles, err := writeTar(files, u.config)
	if err != nil {
		return errors.Trace(err)
	}

	// If we didn't write any files, we are done already
	if len(writtenFiles) == 0 {
		return nil
	}

	// Open the archive
	f, err := os.Open(filename)
	if err != nil {
		return errors.Trace(err)
	}

	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return errors.Trace(err)
	}

	// Print changes
	if u.config.Verbose || len(writtenFiles) <= 3 {
		for _, c := range writtenFiles {
			if c.IsDirectory {
				u.config.Logf("[Upstream] Create Folder %s", c.Name)
			} else {
				u.config.Logf("[Upstream] Create File %s", c.Name)
			}
		}
	}

	return u.uploadArchive(f, strconv.Itoa(int(stat.Size())), writtenFiles)
}

func (u *upstream) uploadArchive(file *os.File, fileSize string, writtenFiles map[string]*fileInformation) error {
	u.config.fileIndex.fileMapMutex.Lock()
	defer u.config.fileIndex.fileMapMutex.Unlock()
	defer file.Close()

	u.config.Logf("[Upstream] Upload %d create changes (size %s)", len(writtenFiles), fileSize)

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

					tar xzpf "$tmpFile" -C '` + u.config.DestPath + `/.' 2>/tmp/devspace-upstream-error;
					echo "` + EndAck + `";
		` // We need that extra new line or otherwise the command is not sent

	// Write command
	_, err := u.stdinPipe.Write([]byte(cmd))
	if err != nil {
		return errors.Trace(err)
	}

	// Wait till confirmation
	err = waitTill(StartAck, u.stdoutPipe)
	if err != nil {
		return errors.Trace(err)
	}

	// Apply rate limit if specified
	var uploadWriter io.Writer = u.stdinPipe
	if u.config.UpstreamLimit > 0 {
		uploadWriter = ratelimit.Writer(u.stdinPipe, ratelimit.NewBucketWithRate(float64(u.config.UpstreamLimit), u.config.UpstreamLimit))
	}

	// Send file through stdin to remote
	_, err = io.Copy(uploadWriter, file)
	if err != nil {
		return errors.Trace(err)
	}

	// Do not remove this line otherwise the delete will fail
	file.Close()

	// Delete local file
	err = os.Remove(file.Name())
	if err != nil {
		return errors.Trace(err)
	}

	// Wait till receive confirmation
	err = waitTill(EndAck, u.stdoutPipe)
	if err != nil {
		return errors.Trace(err)
	}

	// Update sync filemap
	for _, element := range writtenFiles {
		u.config.fileIndex.CreateDirInFileMap(path.Dir(element.Name))
		u.config.fileIndex.fileMap[element.Name] = element
	}

	return nil
}

func (u *upstream) applyRemoves(files []*fileInformation) error {
	u.config.fileIndex.fileMapMutex.Lock()
	defer u.config.fileIndex.fileMapMutex.Unlock()

	u.config.Logf("[Upstream] Handling %d removes", len(files))

	fileMap := u.config.fileIndex.fileMap

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

				// Print changes
				if u.config.Verbose || len(files) <= 3 {
					u.config.Logf("[Upstream] Remove %s", relativePath)
				}
			}
		}

		if removeArguments > 0 {
			rmCommand += " >/dev/null 2>/dev/null && printf \"" + EndAck + "\" || printf \"" + EndAck + "\"\n"

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
