package downloader

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

type Downloader interface {
	EnsureCLI(command, installPath, installFromURL string) (string, error)
}

type downloader struct {
	httpGet getRequest
	isValid IsValid
	install Install
	log     logpkg.Logger
}

type IsValid func(string) (bool, error)
type Install func(archiveFile, installPath, installFromURL string) error

func NewDownloader(install Install, isValid IsValid, log logpkg.Logger) Downloader {
	return &downloader{
		httpGet: http.Get,
		install: install,
		isValid: isValid,
		log:     log,
	}
}

func (d *downloader) EnsureCLI(command, installPath, installFromURL string) (string, error) {
	valid, err := d.isValid(command)
	if err != nil {
		return "", err
	} else if valid {
		return command, nil
	}

	valid, err = d.isValid(installPath)
	if err != nil {
		return "", err
	} else if valid {
		return installPath, nil
	}

	return installPath, d.downloadExecutable(command, installPath, installFromURL)
}

func (d *downloader) downloadExecutable(command, installPath, installFromURL string) error {
	err := os.MkdirAll(filepath.Dir(installPath), 0755)
	if err != nil {
		return err
	}

	err = d.downloadFile(command, installPath, installFromURL)
	if err != nil {
		return errors.Wrap(err, "download file")
	}

	err = os.Chmod(installPath, 0755)
	if err != nil {
		return errors.Wrap(err, "cannot make file executable")
	}

	return nil
}

type getRequest func(url string) (*http.Response, error)

func (d *downloader) downloadFile(command, installPath, installFromURL string) error {
	d.log.StartWait("Downloading " + command + "...")
	defer d.log.StopWait()

	t, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(t)

	archiveFile := filepath.Join(t, "download")
	f, err := os.Create(archiveFile)
	if err != nil {
		return err
	}
	defer f.Close()

	resp, err := d.httpGet(installFromURL)
	if err != nil {
		return errors.Wrap(err, "get url")
	}

	defer resp.Body.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return errors.Wrap(err, "download file")
	}

	err = f.Close()
	if err != nil {
		return err
	}

	// install the file
	return d.install(archiveFile, installPath, installFromURL)
}
