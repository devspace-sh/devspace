package downloader

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/loft-sh/devspace/pkg/util/downloader/commands"

	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

type Downloader interface {
	EnsureCommand() (string, error)
}

type downloader struct {
	httpGet getRequest
	command commands.Command
	log     logpkg.Logger
}

func NewDownloader(command commands.Command, log logpkg.Logger) Downloader {
	return &downloader{
		httpGet: http.Get,
		command: command,
		log:     log,
	}
}

func (d *downloader) EnsureCommand() (string, error) {
	command := d.command.Name()
	valid, err := d.command.IsValid(command)
	if err != nil {
		return "", err
	} else if valid {
		return command, nil
	}

	installPath, err := d.command.InstallPath()
	if err != nil {
		return "", err
	}

	valid, err = d.command.IsValid(installPath)
	if err != nil {
		return "", err
	} else if valid {
		return installPath, nil
	}

	return installPath, d.downloadExecutable(command, installPath, d.command.DownloadURL())
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
	return d.command.Install(archiveFile)
}
