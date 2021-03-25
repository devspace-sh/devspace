package generic

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/helm/downloader"
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/loft-sh/devspace/pkg/util/extract"
	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/mitchellh/go-homedir"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
)

const stableChartRepo = "https://kubernetes-charts.storage.googleapis.com"

type VersionedClient interface {
	IsValidHelm(path string) (bool, error)
	IsInCluster() bool
	KubeContext() string
	Command() string
	DownloadURL() string
}

type Client interface {
	Exec(args []string, helmConfig *latest.HelmConfig) ([]byte, error)
	FetchChart(helmConfig *latest.HelmConfig) (bool, string, error)
	WriteValues(values map[interface{}]interface{}) (string, error)
}

func NewGenericClient(versionedClient VersionedClient, log log.Logger) Client {
	c := &client{
		exec:            command.NewStreamCommand,
		versionedClient: versionedClient,
		extract:         extract.NewExtractor(),
	}

	c.downloader = downloader.NewDownloader(c.installHelmClient, c.versionedClient.IsValidHelm, log)
	return c
}

type client struct {
	exec            command.Exec
	versionedClient VersionedClient
	extract         extract.Extract
	downloader      downloader.Downloader

	helmPath string
}

func (c *client) WriteValues(values map[interface{}]interface{}) (string, error) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	defer f.Close()
	out, err := yaml.Marshal(values)
	if err != nil {
		return "", errors.Wrap(err, "marshal values")
	}

	_, err = f.Write(out)
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}

func (c *client) Exec(args []string, helmConfig *latest.HelmConfig) ([]byte, error) {
	err := c.ensureHelmBinary(helmConfig)
	if err != nil {
		return nil, err
	}

	if c.versionedClient.IsInCluster() == false {
		args = append(args, "--kube-context", c.versionedClient.KubeContext())
	}
	result, err := c.exec(c.helmPath, args).Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("error during '%s %s': %s%s => %v", c.helmPath, strings.Join(args, " "), string(result), string(exitError.Stderr), err)
		}

		return nil, fmt.Errorf("error during '%s %s': %s => %v", c.helmPath, strings.Join(args, " "), string(result), err)
	}

	return result, nil
}

func (c *client) ensureHelmBinary(helmConfig *latest.HelmConfig) error {
	if c.helmPath != "" {
		return nil
	}

	if helmConfig != nil && helmConfig.Path != "" {
		valid, err := c.versionedClient.IsValidHelm(helmConfig.Path)
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

	installPath := filepath.Join(home, constants.DefaultHomeDevSpaceFolder, "bin", c.versionedClient.Command())
	url := c.versionedClient.DownloadURL()
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

func (c *client) FetchChart(helmConfig *latest.HelmConfig) (bool, string, error) {
	chartName, chartRepo := ChartNameAndRepo(helmConfig)
	if chartRepo == "" {
		return false, chartName, nil
	}

	tempFolder, err := ioutil.TempDir("", "")
	if err != nil {
		return false, "", err
	}

	args := []string{"fetch", chartName, "--repo", chartRepo, "--untar", "--untardir", tempFolder}
	if helmConfig.Chart.Version != "" {
		args = append(args, "--version", helmConfig.Chart.Version)
	}
	if helmConfig.Chart.Username != "" {
		args = append(args, "--username", helmConfig.Chart.Username)
	}
	if helmConfig.Chart.Password != "" {
		args = append(args, "--password", helmConfig.Chart.Password)
	}
	if helmConfig.V2 == false {
		args = append(args, "--repository-config=''")
	}

	args = append(args, helmConfig.FetchArgs...)
	out, err := c.Exec(args, helmConfig)
	if err != nil {
		_ = os.RemoveAll(tempFolder)
		return false, "", fmt.Errorf("error running helm fetch: %s => %v", string(out), err)
	}

	return true, filepath.Join(tempFolder, chartName), nil
}

func ChartNameAndRepo(helmConfig *latest.HelmConfig) (string, string) {
	chartName := strings.TrimSpace(helmConfig.Chart.Name)
	chartRepo := helmConfig.Chart.RepoURL
	if strings.HasPrefix(chartName, "stable/") && chartRepo == "" {
		chartName = chartName[7:]
		chartRepo = stableChartRepo
	}

	return chartName, chartRepo
}
