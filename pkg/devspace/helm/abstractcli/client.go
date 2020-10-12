package abstractcli

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/downloader"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/command"
	"github.com/devspace-cloud/devspace/pkg/util/extract"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mitchellh/go-homedir"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
)

type releaseOutputParser func(out []byte) ([]*types.Release, error)

type Client interface {
	InstallChart(releaseName string, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (*types.Release, error)
	Template(releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig, fetchCmd string, getArgs func(chartDir, releaseNamespace, file, context, tillerNamespace string) []string) (string, error)
	DeleteRelease(releaseName string, releaseNamespace string, helmConfig *latest.HelmConfig, deleteCmd string, extraArgs []string) error
	ListReleases(helmConfig *latest.HelmConfig) ([]*types.Release, error)
}

// Client is an abstract cli client to avoid code redundance between v2cli and v3cli
type client struct {
	exec               command.Exec
	downloader         downloader.Downloader
	parseReleaseOutput releaseOutputParser

	config *latest.Config

	kubeClient      kubectl.Client
	tillerNamespace string

	helmPath      string
	helmDownload  string
	versionPrefix string
	useTiller     bool

	log log.Logger
}

// NewClient creates a new helm client
func NewClient(config *latest.Config, kubeClient kubectl.Client, tillerNamespace, versionPrefix, helmDownload string, parseReleaseOutput releaseOutputParser, log log.Logger) (Client, error) {
	c := &client{
		config: config,

		kubeClient:      kubeClient,
		tillerNamespace: tillerNamespace,
		versionPrefix:   versionPrefix,
		helmDownload:    helmDownload,

		useTiller: tillerNamespace != "",

		exec:               command.Command,
		parseReleaseOutput: parseReleaseOutput,
		log:                log,
	}
	c.downloader = downloader.NewDownloader(c.helmInstall, c.isValidHelm, log)
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
			return fmt.Errorf("helm binary at '%s' is not a valid helm %s binary", helmConfig.Path, c.versionPrefix)
		}

		c.helmPath = helmConfig.Path
		return nil
	}

	home, err := homedir.Dir()
	if err != nil {
		return err
	}

	installPath := filepath.Join(home, constants.DefaultHomeDevSpaceFolder, "bin", "helm"+c.versionPrefix)
	url := c.helmDownload
	if runtime.GOOS == "windows" {
		url += ".zip"
		installPath += ".exe"
	} else {
		url += ".tar.gz"
	}

	c.helmPath, err = c.downloader.EnsureCLI("helm"+c.versionPrefix, installPath, url)
	return err
}

func (c *client) helmInstall(archiveFile, installPath, installFromURL string) error {
	t := filepath.Dir(archiveFile)

	// Extract the binary
	extractor := extract.NewExtractor()
	if strings.HasSuffix(installFromURL, ".tar.gz") {
		err := extractor.UntarGz(archiveFile, t)
		if err != nil {
			return errors.Wrap(err, "extract tar.gz")
		}
	} else if strings.HasSuffix(installFromURL, ".zip") {
		err := extractor.Unzip(archiveFile, t)
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

	return strings.HasPrefix(string(out), `Client: &version.Version{SemVer:"`+c.versionPrefix), nil
}
