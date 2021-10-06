package shell

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/helm/downloader"
	v3 "github.com/loft-sh/devspace/pkg/devspace/helm/v3"
	"github.com/loft-sh/devspace/pkg/util/extract"

	"github.com/mitchellh/go-homedir"
)

func devSpaceDefaultBin() string {
	home, _ := homedir.Dir()
	return filepath.Join(home, constants.DefaultHomeDevSpaceFolder, "bin")
}

func isCommandValid(command string) (bool, error) {
	_, err := exec.Command(command, "version").Output()
	if err != nil {
		return false, nil
	}
	return true, nil
}

func installKubectlCommand() (string, error) {
	log := log.GetInstance()
	isValidKubectl := func(command string) (bool, error) {
		return isCommandValid(command)
	}

	installPath := filepath.Join(devSpaceDefaultBin(), "kubectl")
	url := kubectl.KubectlDownload
	if runtime.GOOS == "windows" {
		url += ".exe"
		installPath += ".exe"
	}

	cmdPath, err := downloader.NewDownloader(kubectl.InstallKubectl, isValidKubectl, log).EnsureCLI("kubectl", installPath, url)
	if err != nil {
		return "", err
	}
	return cmdPath, nil
}

func installHelmCommand() (string, error) {
	log := log.GetInstance()
	client := &v3.Client{}

	isValidHelm := func(command string) (bool, error) {
		return isCommandValid(command)
	}

	installPath := filepath.Join(devSpaceDefaultBin(), "helm")
	url := client.DownloadURL()
	if runtime.GOOS == "windows" {
		url += ".zip"
		installPath += ".exe"
	} else {
		url += ".tar.gz"
	}

	cmdPath, err := downloader.NewDownloader(installHelmClient, isValidHelm, log).EnsureCLI("helm", installPath, url)
	if err != nil {
		return "", err
	}
	return cmdPath, nil
}

func installHelmClient(archiveFile, installPath, installFromURL string) error {
	t := filepath.Dir(archiveFile)
	ext := extract.NewExtractor()
	// Extract the binary
	if strings.HasSuffix(installFromURL, ".tar.gz") {
		err := ext.UntarGz(archiveFile, t)
		if err != nil {
			return errors.Wrap(err, "extract tar.gz")
		}
	} else if strings.HasSuffix(installFromURL, ".zip") {
		err := ext.Unzip(archiveFile, t)
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
