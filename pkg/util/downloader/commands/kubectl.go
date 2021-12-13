package commands

import (
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/mitchellh/go-homedir"
	"github.com/otiai10/copy"
)

var (
	kubectlVersion = func() string {
		res, err := http.Get("https://storage.googleapis.com/kubernetes-release/release/stable.txt")
		if err != nil {
			log.Fatal(err)
		}
		content, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		return string(content)
	}
	kubectlDownload = "https://storage.googleapis.com/kubernetes-release/release/" + kubectlVersion() + "/bin/" + runtime.GOOS + "/" + runtime.GOARCH + "/kubectl"
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
	url := kubectlDownload
	if runtime.GOOS == "windows" {
		url += ".exe"
	}

	return url
}

func (k *kubectlCommand) IsValid(path string) (bool, error) {
	out, err := command.NewStreamCommand(path, []string{"version", "--client"}).Output()
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
