package commands

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/loft-sh/devspace/pkg/util/extract"
	"github.com/mitchellh/go-homedir"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	helmVersion  = "v3.6.2"
	helmDownload = "https://get.helm.sh/helm-" + helmVersion + "-" + runtime.GOOS + "-" + runtime.GOARCH
)

func NewHelmV3Command() Command {
	return &helmv3{}
}

type helmv3 struct{}

func (h *helmv3) Name() string {
	return "helm"
}

func (h *helmv3) InstallPath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	installPath := filepath.Join(home, constants.DefaultHomeDevSpaceFolder, "bin", h.Name())
	if runtime.GOOS == "windows" {
		installPath += ".exe"
	}

	return installPath, nil
}

func (h *helmv3) DownloadURL() string {
	url := helmDownload + ".tar.gz"
	if runtime.GOOS == "windows" {
		url = helmDownload + ".zip"
	}

	return url
}

func (h *helmv3) IsValid(path string) (bool, error) {
	out, err := command.NewStreamCommand(path, []string{"version"}).Output()
	if err != nil {
		return false, nil
	}

	return strings.Contains(string(out), `:"v3.`), nil
}

func (h *helmv3) Install(archiveFile string) error {
	installPath, err := h.InstallPath()
	if err != nil {
		return err
	}

	return installHelmBinary(extract.NewExtractor(), archiveFile, installPath, h.DownloadURL())
}
