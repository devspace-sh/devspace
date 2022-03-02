package generic

import (
	"context"
	"fmt"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/util/command"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/downloader"
	"github.com/loft-sh/devspace/pkg/util/downloader/commands"
	"github.com/loft-sh/devspace/pkg/util/extract"
	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/pkg/errors"
)

const stableChartRepo = "https://charts.helm.sh/stable"

type VersionedClient interface {
	Command() commands.Command
}

type Client interface {
	Exec(ctx *devspacecontext.Context, args []string, helmConfig *latest.HelmConfig) ([]byte, error)
	FetchChart(ctx *devspacecontext.Context, helmConfig *latest.HelmConfig) (bool, string, error)
	WriteValues(values map[string]interface{}) (string, error)
}

func NewGenericClient(versionedClient VersionedClient, log log.Logger) Client {
	c := &client{
		log:             log,
		versionedClient: versionedClient,
		extract:         extract.NewExtractor(),
	}

	c.downloader = downloader.NewDownloader(versionedClient.Command(), log)
	return c
}

type client struct {
	log             log.Logger
	versionedClient VersionedClient
	extract         extract.Extract
	downloader      downloader.Downloader

	helmPath string
}

func (c *client) WriteValues(values map[string]interface{}) (string, error) {
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

func (c *client) Exec(ctx *devspacecontext.Context, args []string, helmConfig *latest.HelmConfig) ([]byte, error) {
	err := c.ensureHelmBinary(ctx.Context, helmConfig)
	if err != nil {
		return nil, err
	}

	if !ctx.KubeClient.IsInCluster() {
		args = append(args, "--kube-context", ctx.KubeClient.CurrentContext())
	}

	// disable log for list, because it prints same command multiple times if we've multiple deployments.
	if args[0] != "list" {
		c.log.Infof("Execute '%s %s'", c.helmPath, strings.Join(args, " "))
	}
	result, err := command.Output(ctx.Context, ctx.WorkingDir, c.helmPath, args...)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("error during '%s %s':\n %s%s => %v", c.helmPath, strings.Join(args, " "), string(result), string(exitError.Stderr), err)
		}

		return nil, fmt.Errorf("error during '%s %s':\n %s => %v %s", c.helmPath, strings.Join(args, " "), string(result), err)
	}

	return result, nil
}

func (c *client) ensureHelmBinary(ctx context.Context, helmConfig *latest.HelmConfig) error {
	if c.helmPath != "" {
		return nil
	}

	if helmConfig != nil && helmConfig.Path != "" {
		valid, err := c.versionedClient.Command().IsValid(ctx, helmConfig.Path)
		if err != nil {
			return err
		} else if !valid {
			return fmt.Errorf("helm binary at '%s' is not a valid helm v2 binary", helmConfig.Path)
		}

		c.helmPath = helmConfig.Path
		return nil
	}

	var err error
	c.helmPath, err = c.downloader.EnsureCommand(ctx)
	return err
}

func (c *client) FetchChart(ctx *devspacecontext.Context, helmConfig *latest.HelmConfig) (bool, string, error) {
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
	args = append(args, "--repository-config=''")

	args = append(args, helmConfig.FetchArgs...)
	out, err := c.Exec(ctx, args, helmConfig)
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
