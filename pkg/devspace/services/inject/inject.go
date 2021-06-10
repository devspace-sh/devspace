package inject

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/loft-sh/devspace/assets"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"github.com/loft-sh/devspace/pkg/util/hash"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// DevSpaceHelperBaseURL is the base url where to look for the sync helper
const DevSpaceHelperBaseURL = "https://github.com/loft-sh/devspace/releases"

// DevSpaceHelperTempFolder is the local folder where we store the sync helper
const DevSpaceHelperTempFolder = "devspacehelper"

// helperBinaryRegEx is the regexp that finds the correct download link for the sync helper binary
var helperBinaryRegEx = `href="(\/loft-sh\/devspace\/releases\/download\/[^\/]*\/%s)"`

// DevSpaceHelperContainerPath is the path of the devspace helper in the container
const DevSpaceHelperContainerPath = "/tmp/devspacehelper"

// InjectDevSpaceHelper injects the devspace helper into the provided container
func InjectDevSpaceHelper(client kubectl.Client, pod *v1.Pod, container string, arch string, log logpkg.Logger) error {
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
		// check if we can find it in the assets
		helperBytes, err := assets.Asset("release/" + localHelperName)
		if err == nil {
			return injectSyncHelperFromBytes(client, pod, container, helperBytes)
		}

		homedir, err := homedir.Dir()
		if err != nil {
			return err
		}

		syncBinaryFolder := filepath.Join(homedir, constants.DefaultHomeDevSpaceFolder, DevSpaceHelperTempFolder, version)

		// Download sync helper if necessary
		err = downloadSyncHelper(localHelperName, syncBinaryFolder, version, log)
		if err != nil {
			return errors.Wrap(err, "download devspace helper")
		}

		// Inject sync helper
		filepath := filepath.Join(syncBinaryFolder, localHelperName)
		err = injectSyncHelper(client, pod, container, filepath)
		if err != nil {
			return errors.Wrap(err, "inject devspace helper")
		}
	}

	return nil
}

func StartStream(client kubectl.Client, pod *v1.Pod, container string, command []string, reader io.Reader, writer io.Writer) error {
	stderrBuffer := &bytes.Buffer{}
	err := client.ExecStream(&kubectl.ExecStreamOptions{
		Pod:       pod,
		Container: container,
		Command:   command,
		Stdin:     reader,
		Stdout:    writer,
		Stderr:    stderrBuffer,
	})
	if err != nil {
		return fmt.Errorf("%s %v", stderrBuffer.String(), err)
	}
	return nil
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
	url := ""
	if version == "latest" {
		url = fmt.Sprintf("%s/%s", DevSpaceHelperBaseURL, version)
	} else {
		url = fmt.Sprintf("%s/tag/%s", DevSpaceHelperBaseURL, version)
	}

	// Download html
	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "get url")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "read body")
	}

	regEx, err := regexp.Compile(fmt.Sprintf(helperBinaryRegEx, filename))
	if err != nil {
		return err
	}

	matches := regEx.FindStringSubmatch(string(body))
	if len(matches) != 2 {
		return errors.Errorf("couldn't find %s in github release %s at url %s", filename, version, url)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return errors.Wrap(err, "create filepath")
	}
	defer out.Close()

	resp, err = http.Get("https://github.com" + matches[1])
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

func injectSyncHelperFromBytes(client kubectl.Client, pod *v1.Pod, container string, b []byte) error {
	// Compress the sync helper and then copy it to the container
	reader, writer := io.Pipe()

	defer reader.Close()
	defer writer.Close()

	// Start reading on the other end
	errChan := make(chan error)
	go func() {
		errChan <- client.CopyFromReader(pod, container, "/tmp", reader)
	}()

	// Use compression
	gw := gzip.NewWriter(writer)
	defer gw.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	hdr, err := tar.FileInfoHeader(helperFileInfo(b), DevSpaceHelperTempFolder)
	if err != nil {
		return errors.Wrap(err, "create tar file info header")
	}

	hdr.Name = "devspacehelper"

	// Set permissions correctly
	hdr.Mode = 0777
	hdr.Uid = 0
	hdr.Uname = "root"
	hdr.Gid = 0
	hdr.Gname = "root"

	if err := tarWriter.WriteHeader(hdr); err != nil {
		return errors.Wrap(err, "tar write header")
	}

	go func() {
		_, err = io.Copy(tarWriter, bytes.NewReader(b))

		// Close all writers and file
		tarWriter.Close()
		gw.Close()
		writer.Close()
		
		errChan <- err
	}()
	
	err = <-errChan
	if err != nil {
		return errors.Wrap(err, "inject devspacehelper")
	}
	
	return <-errChan
}

func injectSyncHelper(client kubectl.Client, pod *v1.Pod, container string, filepath string) error {
	// Compress the sync helper and then copy it to the container
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()

	// Start reading on the other end
	errChan := make(chan error)
	go func() {
		errChan <- client.CopyFromReader(pod, container, "/tmp", reader)
	}()

	// Use compression
	gw := gzip.NewWriter(writer)
	defer gw.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

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

	hdr, err := tar.FileInfoHeader(stat, filepath)
	if err != nil {
		return errors.Wrap(err, "create tar file info header")
	}

	hdr.Name = "devspacehelper"

	// Set permissions correctly
	hdr.Mode = 0777
	hdr.Uid = 0
	hdr.Uname = "root"
	hdr.Gid = 0
	hdr.Gname = "root"

	if err := tarWriter.WriteHeader(hdr); err != nil {
		return errors.Wrap(err, "tar write header")
	}

	go func() {
		_, err = io.Copy(tarWriter, f)

		// Close all writers and file
		f.Close()
		tarWriter.Close()
		gw.Close()
		writer.Close()

		errChan <- err
	}()

	err = <-errChan
	if err != nil {
		return errors.Wrap(err, "inject devspacehelper")
	}

	return <-errChan
}
