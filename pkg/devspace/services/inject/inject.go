package inject

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/loft-sh/devspace/assets"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"github.com/loft-sh/devspace/pkg/util/hash"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

// DevSpaceHelperBaseURL is the base url where to look for the sync helper
const DevSpaceHelperBaseURL = "https://github.com/loft-sh/devspace/releases"

// DevSpaceHelperTempFolder is the local folder where we store the sync helper
const DevSpaceHelperTempFolder = "devspacehelper"

// helperBinaryRegEx is the regexp that finds the correct download link for the sync helper binary
var helperBinaryRegEx = `href="(\/loft-sh\/devspace\/releases\/download\/[^\/]*\/%s)"`

// DevSpaceHelperContainerPath is the path of the devspace helper in the container
const DevSpaceHelperContainerPath = "/tmp/devspacehelper"

// injectMutex makes sure we only inject one devspacehelper at the time
var injectMutex = sync.Mutex{}

// InjectDevSpaceHelper injects the devspace helper into the provided container
func InjectDevSpaceHelper(client kubectl.Client, pod *v1.Pod, container string, arch string, log logpkg.Logger) error {
	if log == nil {
		log = logpkg.Discard
	}

	injectMutex.Lock()
	defer injectMutex.Unlock()

	// Compare sync versions
	version := upgrade.GetRawVersion()
	if version == "" {
		version = "latest"
	}
	if arch != "" {
		if latest.ContainerArchitecture(arch) == latest.ContainerArchitectureAmd64 {
			arch = ""
		} else {
			arch = "-" + arch
		}
	}

	// Check if sync is already in pod
	localHelperName := "devspacehelper" + arch
	stdout, _, err := client.ExecBuffered(pod, container, []string{DevSpaceHelperContainerPath, "version"}, nil)
	if err != nil || version != string(stdout) {
		homedir, err := homedir.Dir()
		if err != nil {
			return err
		}

		syncBinaryFolder := filepath.Join(homedir, constants.DefaultHomeDevSpaceFolder, DevSpaceHelperTempFolder, version)
		if os.Getenv("DEVSPACE_INJECT_LOCAL") != "true" {
			// Install devspacehelper inside container
			log.Infof("Trying to download devspacehelper into pod %s/%s", pod.Namespace, pod.Name)
			err = installDevSpaceHelperInContainer(client, pod, container, version, localHelperName)
			if err == nil {
				log.Donef("Successfully injected devspacehelper into pod %s/%s", pod.Namespace, pod.Name)
				return nil
			}

			log.Warnf("Couldn't download devspacehelper in container, error: %s", err)
		}

		log.Info("Trying to inject devspacehelper from local machine")

		// check if we can find it in the assets
		helperBytes, err := assets.Asset("release/" + localHelperName)
		if err == nil {
			return injectSyncHelperFromBytes(client, pod, container, helperFileInfo(helperBytes), bytes.NewReader(helperBytes))
		}

		// Download sync helper if necessary
		err = downloadSyncHelper(localHelperName, syncBinaryFolder, version, log)
		if err != nil {
			return errors.Wrap(err, "download devspace helper")
		}

		// Inject sync helper
		err = injectSyncHelper(client, pod, container, filepath.Join(syncBinaryFolder, localHelperName))
		if err != nil {
			return errors.Wrap(err, "inject devspace helper")
		}

		log.Donef("Successfully injected devspacehelper into pod %s/%s", pod.Namespace, pod.Name)
		return nil
	}

	return nil
}

func installDevSpaceHelperInContainer(client kubectl.Client, pod *v1.Pod, container, version, filename string) error {
	url, err := devSpaceHelperDownloadURL(version, filename)
	if err != nil {
		return err
	}

	curl := fmt.Sprintf("curl -L %s -o %s", url, DevSpaceHelperContainerPath)
	chmod := fmt.Sprintf("chmod +x %s", DevSpaceHelperContainerPath)
	cmd := curl + " && " + chmod

	_, _, err = client.ExecBuffered(pod, container, []string{"sh", "-c", cmd}, nil)
	if err != nil {
		return err
	}

	stdout, _, err := client.ExecBuffered(pod, container, []string{DevSpaceHelperContainerPath, "version"}, nil)
	if err != nil {
		return err
	}

	if version != string(stdout) {
		return fmt.Errorf("devspacehelper(%s) and devspace(%s) differs in version", string(stdout), version)
	}

	return nil
}

// getDownloadURL
func devSpaceHelperDownloadURL(version, filename string) (string, error) {
	url := ""
	if version == "latest" {
		url = fmt.Sprintf("%s/%s", DevSpaceHelperBaseURL, version)
	} else {
		url = fmt.Sprintf("%s/tag/%s", DevSpaceHelperBaseURL, version)
	}

	// Download html
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.Wrap(err, "get url")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "read body")
	}

	regEx, err := regexp.Compile(fmt.Sprintf(helperBinaryRegEx, filename))
	if err != nil {
		return "", err
	}

	matches := regEx.FindStringSubmatch(string(body))
	if len(matches) != 2 {
		return "", errors.Errorf("couldn't find %s in github release %s at url %s", filename, version, url)
	}
	return "https://github.com" + matches[1], nil
}

func downloadSyncHelper(helperName, syncBinaryFolder, version string, log logpkg.Logger) error {
	filepath := filepath.Join(syncBinaryFolder, helperName)

	// Check if file exists
	_, err := os.Stat(filepath)
	if err == nil {
		// make sure the sha is correct, but skip for latest because that is development
		if version == "latest" {
			return nil
		}

		// download sha256 html
		url := fmt.Sprintf("https://github.com/loft-sh/devspace/releases/download/%s/%s.sha256", version, helperName)
		resp, err := http.Get(url)
		if err != nil {
			log.Warnf("Couldn't retrieve helper sha256: %v", err)
			return nil
		}

		shaHash, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Warnf("Couldn't read helper sha256 request: %v", err)
			return nil
		}

		// hash the local binary
		fileHash, err := hash.File(filepath)
		if err != nil {
			log.Warnf("Couldn't hash local helper binary: %v", err)
			return nil
		}

		// the file is correct we skip downloading
		if fileHash == strings.Split(string(shaHash), " ")[0] {
			return nil
		}

		// remove the old binary
		err = os.Remove(filepath)
		if err != nil {
			return errors.Wrap(err, "remove corrupt helper binary")
		}
	}

	// Make sync binary
	log.Infof("Couldn't find %s, will try to download it now", helperName)
	err = os.MkdirAll(syncBinaryFolder, 0755)
	if err != nil {
		return errors.Wrap(err, "mkdir helper binary folder")
	}
	return downloadFile(version, filepath, helperName)
}

func downloadFile(version string, filepath string, filename string) error {
	// Create download url
	url, err := devSpaceHelperDownloadURL(version, filename)
	if err != nil {
		return errors.Wrap(err, "find download URL")
	}

	out, err := os.Create(filepath)
	if err != nil {
		return errors.Wrap(err, "create filepath")
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "download devspace helper")
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return errors.Wrap(err, "download devspace helper to file")
	}

	return nil
}

type helperFileInfo []byte

func (h helperFileInfo) Name() string {
	return DevSpaceHelperTempFolder
}
func (h helperFileInfo) Size() int64 {
	return int64(len([]byte(h)))
}
func (h helperFileInfo) Mode() os.FileMode {
	return 0777
}
func (h helperFileInfo) ModTime() time.Time {
	return time.Now()
}
func (h helperFileInfo) IsDir() bool {
	return false
}
func (h helperFileInfo) Sys() interface{} {
	return nil
}

func injectSyncHelper(client kubectl.Client, pod *v1.Pod, container string, filepath string) error {
	// Stat sync helper
	stat, err := os.Stat(filepath)
	if err != nil {
		return errors.Wrap(err, "stat sync helper")
	}

	// Open file
	f, err := os.Open(filepath)
	if err != nil {
		return errors.Wrap(err, "open file")
	}

	defer f.Close()
	return injectSyncHelperFromBytes(client, pod, container, stat, f)
}

func injectSyncHelperFromBytes(client kubectl.Client, pod *v1.Pod, container string, fi fs.FileInfo, bytesReader io.Reader) error {
	writerComplete := make(chan struct{})
	readerComplete := make(chan struct{})

	// Compress the sync helper and then copy it to the container
	reader, writer := io.Pipe()
	var (
		retErr    error
		setRetErr sync.Once
	)

	go func() {
		defer close(readerComplete)
		defer func() {
			if r := recover(); r != nil {
				setRetErr.Do(func() {
					retErr = fmt.Errorf("%v", r)
				})
			}
		}()
		defer reader.Close()

		err := client.CopyFromReader(pod, container, "/tmp", reader)
		setRetErr.Do(func() {
			retErr = err
		})
	}()

	go func() {
		defer close(writerComplete)
		defer func() {
			if r := recover(); r != nil {
				setRetErr.Do(func() {
					retErr = fmt.Errorf("%v", r)
				})
			}
		}()
		defer writer.Close()

		// Use compression
		gw := gzip.NewWriter(writer)
		defer gw.Close()

		// Create tar writer
		tarWriter := tar.NewWriter(gw)
		defer tarWriter.Close()

		hdr, err := tar.FileInfoHeader(fi, DevSpaceHelperTempFolder)
		if err != nil {
			setRetErr.Do(func() {
				retErr = err
			})
			return
		}

		hdr.Name = "devspacehelper"

		// Set permissions correctly
		hdr.Mode = 0777
		hdr.Uid = 0
		hdr.Uname = "root"
		hdr.Gid = 0
		hdr.Gname = "root"

		err = tarWriter.WriteHeader(hdr)
		if err != nil {
			setRetErr.Do(func() {
				retErr = err
			})
			return
		}

		_, err = io.Copy(tarWriter, bytesReader)
		setRetErr.Do(func() {
			retErr = err
		})
	}()

	// wait for reader or writer to finish
	select {
	case <-writerComplete:
		if retErr != nil {
			return retErr
		}

		<-readerComplete
	case <-readerComplete:
		if retErr != nil {
			return retErr
		}

		<-writerComplete
	}

	return nil
}
