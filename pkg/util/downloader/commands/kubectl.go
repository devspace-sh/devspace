package commands

import (
	"context"
	"io/ioutil"
	"mvdan.cc/sh/v3/expand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/mitchellh/go-homedir"
	"github.com/otiai10/copy"
)

func NewKubectlCommand() Command {
	return &kubectlCommand{}
}

type kubectlCommand struct{}

func (k *kubectlCommand) Name() string {
	return "kubectl"
}

func (k *kubectlCommand) InstallPath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	installPath := filepath.Join(home, constants.DefaultHomeDevSpaceFolder, "bin", "kubectl")
	if runtime.GOOS == "windows" {
		installPath += ".exe"
	}

	return installPath, nil
}

func (k *kubectlCommand) DownloadURL() string {
	// let the default kubectl version be 1.24.1
	kubectlVersion := "v1.24.1"

	// try to fetch latest kubectl version if it fails use default version
	res, err := http.Get("https://storage.googleapis.com/kubernetes-release/release/stable.txt")
	if err == nil {
		content, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err == nil {
			kubectlVersion = string(content)
		}
	}

	url := "https://storage.googleapis.com/kubernetes-release/release/" + kubectlVersion + "/bin/" + runtime.GOOS + "/" + runtime.GOARCH + "/kubectl"
	if runtime.GOOS == "windows" {
		url += ".exe"
	}

	return url
}

func (k *kubectlCommand) IsValid(ctx context.Context, path string) (bool, error) {
	out, err := command.Output(ctx, "", expand.ListEnviron(os.Environ()...), path, "version", "--client")
	if err != nil {
		return false, nil
	}

	return strings.Contains(string(out), `Client Version`), nil
}

func (k *kubectlCommand) Install(archiveFile string) error {
	installPath, err := k.InstallPath()
	if err != nil {
		return err
	}

	return copy.Copy(archiveFile, installPath)
}
