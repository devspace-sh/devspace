package v2cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/util/command"
	"github.com/devspace-cloud/devspace/pkg/util/extract"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mitchellh/go-homedir"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
)

var (
	helmVersion  = "v2.16.3"
	helmDownload = "https://get.helm.sh/helm-" + helmVersion + "-" + runtime.GOOS + "-amd64"
)

type client struct {
	exec    command.Exec
	extract extract.Extract
	httpGet getRequest

	config *latest.Config

	kubeClient      kubectl.Client
	tillerNamespace string

	helmPath string
	log      log.Logger
}

// NewClient creates a new helm client
func NewClient(config *latest.Config, kubeClient kubectl.Client, tillerNamespace string, log log.Logger) (types.Client, error) {
	if tillerNamespace == "" {
		tillerNamespace = kubeClient.Namespace()
	}

	return &client{
		config: config,

		kubeClient:      kubeClient,
		tillerNamespace: tillerNamespace,

		exec:    command.Command,
		extract: extract.NewExtractor(),
		httpGet: http.Get,

		log: log,
	}, nil
}

func (c *client) ensureHelmBinary(helmConfig *latest.HelmConfig) error {
	if c.helmPath != "" {
		return nil
	}

	if helmConfig != nil && helmConfig.Path != "" {
		if !c.isValidHelm(helmConfig.Path) {
			return fmt.Errorf("Helm binary at '%s' is not a valid helm v2 binary", helmConfig.Path)
		}

		c.helmPath = helmConfig.Path
		return nil
	}

	c.helmPath = "helm"
	if c.isValidHelm(c.helmPath) {
		return nil
	}

	home, err := homedir.Dir()
	if err != nil {
		return err
	}

	c.helmPath = filepath.Join(home, constants.DefaultHomeDevSpaceFolder, "bin", "helm")
	if c.isValidHelm(c.helmPath) {
		return nil
	}

	return c.ensureHelmExecutable(c.helmPath)
}

func (c *client) isValidHelm(path string) bool {
	out, err := c.exec(path, []string{"version", "--client"}).CombinedOutput()
	if err != nil {
		return false
	}

	return strings.HasPrefix(string(out), `Client: &version.Version{SemVer:"v2`)
}

func (c *client) ensureHelmExecutable(path string) error {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	url := helmDownload
	if runtime.GOOS == "windows" {
		url += ".zip"
		path += ".exe"
	} else {
		url += ".tar.gz"
	}

	err = c.downloadFile(path, url)
	if err != nil {
		return errors.Wrap(err, "download helm")
	}

	// make executable
	err = os.Chmod(path, 0755)
	if err != nil {
		return errors.Wrap(err, "cannot make file executable")
	}

	return nil
}

type getRequest func(url string) (*http.Response, error)

func (c *client) downloadFile(target string, url string) error {
	c.log.StartWait("Downloading helm...")
	defer c.log.StopWait()

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

	resp, err := c.httpGet(url)
	if err != nil {
		return errors.Wrap(err, "get url")
	}

	defer resp.Body.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return errors.Wrap(err, "download helm archive")
	}

	err = f.Close()
	if err != nil {
		return err
	}

	// Extract the binary
	if strings.HasSuffix(url, ".tar.gz") {
		err = c.extract.UntarGz(archiveFile, t)
		if err != nil {
			return errors.Wrap(err, "extract tar.gz")
		}
	} else if strings.HasSuffix(url, ".zip") {
		err = c.extract.Unzip(archiveFile, t)
		if err != nil {
			return errors.Wrap(err, "extract zip")
		}
	}

	// Copy file to target location
	if runtime.GOOS == "windows" {
		return copy.Copy(filepath.Join(t, runtime.GOOS+"-amd64", "helm.exe"), target)
	}

	return copy.Copy(filepath.Join(t, runtime.GOOS+"-amd64", "helm"), target)
}
