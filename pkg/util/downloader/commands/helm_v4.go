package commands

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	utilscommand "github.com/loft-sh/utils/pkg/command"
	downloadercommands "github.com/loft-sh/utils/pkg/downloader/commands"
	"github.com/loft-sh/utils/pkg/extract"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/expand"
)

const helmVersion = "v4.0.4"

var helmDownload = "https://get.helm.sh/helm-" + helmVersion + "-" + runtime.GOOS + "-" + runtime.GOARCH

func NewHelmV4Command() downloadercommands.Command {
	return &helmCommand{}
}

type helmCommand struct{}

func (h *helmCommand) Name() string {
	return "helm"
}

func (h *helmCommand) InstallPath(toolHomeFolder string) (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	installPath := filepath.Join(home, toolHomeFolder, "bin", h.Name())
	if runtime.GOOS == "windows" {
		installPath += ".exe"
	}

	return installPath, nil
}

func (h *helmCommand) DownloadURL() string {
	if runtime.GOOS == "windows" {
		return helmDownload + ".zip"
	}

	return helmDownload + ".tar.gz"
}

func (h *helmCommand) IsValid(ctx context.Context, path string) (bool, error) {
	out, err := utilscommand.Output(ctx, "", expand.ListEnviron(os.Environ()...), path, "version")
	if err != nil {
		return false, nil
	}

	return strings.Contains(string(out), `:"v4.`), nil
}

func (h *helmCommand) Install(toolHomeFolder, archiveFile string) error {
	installPath, err := h.InstallPath(toolHomeFolder)
	if err != nil {
		return err
	}

	return installHelmBinary(archiveFile, installPath, h.DownloadURL())
}

func installHelmBinary(archiveFile, installPath, installFromURL string) error {
	targetDir := filepath.Dir(archiveFile)
	extractor := extract.NewExtractor()

	if strings.HasSuffix(installFromURL, ".tar.gz") {
		if err := extractor.UntarGz(archiveFile, targetDir); err != nil {
			return errors.Wrap(err, "extract tar.gz")
		}
	} else if strings.HasSuffix(installFromURL, ".zip") {
		if err := extractor.Unzip(archiveFile, targetDir); err != nil {
			return errors.Wrap(err, "extract zip")
		}
	}

	binaryName := "helm"
	if runtime.GOOS == "windows" {
		binaryName = "helm.exe"
	}

	sourcePath := filepath.Join(targetDir, runtime.GOOS+"-"+runtime.GOARCH, binaryName)
	return copyFile(sourcePath, installPath)
}

func copyFile(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	target, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = target.Close()
	}()

	if _, err := io.Copy(target, source); err != nil {
		return err
	}

	return target.Close()
}
