package commands

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/loft-sh/devspace/pkg/util/extract"
	"github.com/mitchellh/go-homedir"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	helmV2Version  = "v2.17.0"
	helmV2Download = "https://get.helm.sh/helm-" + helmV2Version + "-" + runtime.GOOS + "-" + runtime.GOARCH
)

func NewHelmV2Command() Command {
	return &helmv2{}
}

type helmv2 struct{}

func (h *helmv2) Name() string {
	return "helm"
}

func (h *helmv2) InstallPath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	installPath := filepath.Join(home, constants.DefaultHomeDevSpaceFolder, "bin", "helm2")
	if runtime.GOOS == "windows" {
		installPath += ".exe"
	}

	return installPath, nil
}

func (h *helmv2) DownloadURL() string {
	url := helmV2Download + ".tar.gz"
	if runtime.GOOS == "windows" {
		url = helmV2Download + ".zip"
	}

	return url
}

func (h *helmv2) IsValid(path string) (bool, error) {
	out, err := command.NewStreamCommand(path, []string{"version", "--client"}).Output()
	if err != nil {
		return false, nil
	}

	return strings.Contains(string(out), `:"v2.`), nil
}

func (h *helmv2) Install(archiveFile string) error {
	installPath, err := h.InstallPath()
	if err != nil {
		return err
	}

	return installHelmBinary(extract.NewExtractor(), archiveFile, installPath, h.DownloadURL())
}

func installHelmBinary(extract extract.Extract, archiveFile, installPath, installFromURL string) error {
	t := filepath.Dir(archiveFile)

	// Extract the binary
	if strings.HasSuffix(installFromURL, ".tar.gz") {
		err := extract.UntarGz(archiveFile, t)
		if err != nil {
			return errors.Wrap(err, "extract tar.gz")
		}
	} else if strings.HasSuffix(installFromURL, ".zip") {
		err := extract.Unzip(archiveFile, t)
		if err != nil {
			return errors.Wrap(err, "extract zip")
		}
	}

	// Copy file to target location
	if runtime.GOOS == "windows" {
		return copy.Copy(filepath.Join(t, runtime.GOOS+"-"+runtime.GOARCH, "helm.exe"), installPath)
	}

	return copy.Copy(filepath.Join(t, runtime.GOOS+"-"+runtime.GOARCH, "helm"), installPath)
}
