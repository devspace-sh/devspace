package server

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/loft-sh/devspace/assets"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

// UIDownloadBaseURL is the base url where to look for the ui
const UIDownloadBaseURL = "https://github.com/loft-sh/devspace/releases"

// UIDownloadRegEx is the regexp that finds the correct download link for the ui
var UIDownloadRegEx = regexp.MustCompile(`href="(\/loft-sh\/devspace\/releases\/download\/[^\/]*\/ui.tar.gz)"`)

// UITempFolder is the temp folder to cache the ui in
const UITempFolder = "ui"

func downloadUI() (string, error) {
	// Compare sync versions
	version := upgrade.GetRawVersion()
	if version == "" {
		version = "latest"
	}

	homedir, _ := homedir.Dir()

	// Check if ui was already downloaded / extracted
	uiFolder := filepath.Join(homedir, constants.DefaultHomeDevSpaceFolder, UITempFolder, version)

	// Download / extract if necessary
	err := downloadUITar(uiFolder, version)
	if err != nil {
		return "", errors.Wrap(err, "download ui tar ball")
	}

	return uiFolder, nil
}

func downloadUITar(uiFolder, version string) error {
	// Check if file exists
	_, err := os.Stat(filepath.Join(uiFolder, "index.html"))
	if err == nil {
		return nil
	}

	// Make ui folder
	err = os.MkdirAll(uiFolder, 0755)
	if err != nil {
		return errors.Wrap(err, "mkdir ui folder folder")
	}

	// Download archive
	return downloadFile(version, uiFolder)
}

func downloadFile(version string, folder string) error {
	uiBytes, err := assets.Asset("release/ui.tar.gz")
	if err == nil {
		return untar(bytes.NewReader(uiBytes), folder)
	}

	// Create download url
	url := ""
	if version == "latest" {
		url = fmt.Sprintf("%s/%s", UIDownloadBaseURL, version)
	} else {
		url = fmt.Sprintf("%s/tag/%s", UIDownloadBaseURL, version)
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

	matches := UIDownloadRegEx.FindStringSubmatch(string(body))
	if len(matches) != 2 {
		return errors.Errorf("Couldn't find ui in github release %s at url %s", version, url)
	}

	resp, err = http.Get("https://github.com" + matches[1])
	if err != nil {
		return errors.Wrap(err, "download ui archive")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	return untar(resp.Body, folder)
}

func untar(r io.Reader, dir string) (err error) {
	t0 := time.Now()
	nFiles := 0
	madeDir := map[string]struct{}{}

	zr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("requires gzip-compressed body: %v", err)
	}

	tr := tar.NewReader(zr)
	for {
		f, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar error: %v", err)
		}
		if !validRelPath(f.Name) {
			return fmt.Errorf("tar contained invalid name error %q", f.Name)
		}
		rel := filepath.FromSlash(f.Name)
		abs := filepath.Join(dir, rel)

		fi := f.FileInfo()
		mode := fi.Mode()
		switch {
		case mode.IsRegular():
			dir := filepath.Dir(abs)
			if _, ok := madeDir[dir]; !ok {
				if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
					return err
				}
				madeDir[dir] = struct{}{}
			}
			wf, err := os.OpenFile(abs, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode.Perm())
			if err != nil {
				return err
			}
			n, err := io.Copy(wf, tr)
			if closeErr := wf.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				return fmt.Errorf("error writing to %s: %v", abs, err)
			}
			if n != f.Size {
				return fmt.Errorf("only wrote %d bytes to %s; expected %d", n, abs, f.Size)
			}
			modTime := f.ModTime
			if modTime.After(t0) {
				modTime = t0
			}
			if !modTime.IsZero() {
				_ = os.Chtimes(abs, modTime, modTime)
			}
			nFiles++
		case mode.IsDir():
			if err := os.MkdirAll(abs, 0755); err != nil {
				return err
			}
			madeDir[abs] = struct{}{}
		default:
			return fmt.Errorf("tar file entry %s contained unsupported file type %v", f.Name, mode)
		}
	}
	return nil
}

func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}
