package v2cli

import (
	"fmt"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/v2cli/downloader"
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
	helmVersion  = "v2.16.9"
	helmDownload = "https://get.helm.sh/helm-" + helmVersion + "-" + runtime.GOOS + "-amd64"
)

type client struct {
	exec       command.Exec
	extract    extract.Extract
	downloader downloader.Downloader

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

	c := &client{
		config: config,

		kubeClient:      kubeClient,
		tillerNamespace: tillerNamespace,

		exec:    command.Command,
		extract: extract.NewExtractor(),

		log: log,
	}
	c.downloader = downloader.NewDownloader(c.installHelmClient, c.isValidHelm, log)
	return c, nil
}

func (c *client) ensureHelmBinary(helmConfig *latest.HelmConfig) error {
	if c.helmPath != "" {
		return nil
	}

	if helmConfig != nil && helmConfig.Path != "" {
		valid, err := c.isValidHelm(helmConfig.Path)
		if err != nil {
			return err
		} else if !valid {
			return fmt.Errorf("helm binary at '%s' is not a valid helm v2 binary", helmConfig.Path)
		}

		c.helmPath = helmConfig.Path
		return nil
	}

	home, err := homedir.Dir()
	if err != nil {
		return err
	}

	installPath := filepath.Join(home, constants.DefaultHomeDevSpaceFolder, "bin", "helm")
	url := helmDownload
	if runtime.GOOS == "windows" {
		url += ".zip"
		installPath += ".exe"
	} else {
		url += ".tar.gz"
	}

	c.helmPath, err = c.downloader.EnsureCLI("helm", installPath, url)
	return err
}

func (c *client) installHelmClient(archiveFile, installPath, installFromURL string) error {
	t := filepath.Dir(archiveFile)

	// Extract the binary
	if strings.HasSuffix(installFromURL, ".tar.gz") {
		err := c.extract.UntarGz(archiveFile, t)
		if err != nil {
			return errors.Wrap(err, "extract tar.gz")
		}
	} else if strings.HasSuffix(installFromURL, ".zip") {
		err := c.extract.Unzip(archiveFile, t)
		if err != nil {
			return errors.Wrap(err, "extract zip")
		}
	}

	// Copy file to target location
	if runtime.GOOS == "windows" {
		return copy.Copy(filepath.Join(t, runtime.GOOS+"-amd64", "helm.exe"), installPath)
	}

	return copy.Copy(filepath.Join(t, runtime.GOOS+"-amd64", "helm"), installPath)
}

func (c *client) isValidHelm(path string) (bool, error) {
	out, err := c.exec(path, []string{"version", "--client"}).CombinedOutput()
	if err != nil {
		return false, nil
	}

	return strings.HasPrefix(string(out), `Client: &version.Version{SemVer:"v2`), nil
}
