package v3

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	dependencyutil "github.com/loft-sh/devspace/pkg/devspace/dependency/util"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/helm/generic"
	"github.com/loft-sh/devspace/pkg/devspace/helm/types"
	"github.com/loft-sh/devspace/pkg/util/downloader/commands"
	"github.com/loft-sh/devspace/pkg/util/log"
)

type client struct {
	genericHelm generic.Client
}

// NewClient creates a new helm v3 Client
func NewClient(log log.Logger) (types.Client, error) {
	c := &client{}
	c.genericHelm = generic.NewGenericClient(commands.NewHelmV3Command(), log)
	return c, nil
}

// InstallChart installs the given chart via helm v3
func (c *client) InstallChart(ctx *devspacecontext.Context, releaseName string, releaseNamespace string, values map[string]interface{}, helmConfig *latest.HelmConfig) (*types.Release, error) {
	valuesFile, err := c.genericHelm.WriteValues(values)
	if err != nil {
		return nil, err
	}
	defer os.Remove(valuesFile)

	if releaseNamespace == "" {
		releaseNamespace = ctx.KubeClient.Namespace()
	}

	args := []string{
		"upgrade",
		releaseName,
		"--namespace",
		releaseNamespace,
		"--values",
		valuesFile,
		"--install",
	}

	// Chart settings
	if helmConfig.Chart.Source != nil {
		chartName, err := dependencyutil.DownloadDependency(ctx.Context, ctx.WorkingDir, helmConfig.Chart.Source, ctx.Log)
		if err != nil {
			return nil, err
		}

		chartName = filepath.Dir(chartName)
		args = append(args, chartName)
	} else {
		chartName, chartRepo := generic.ChartNameAndRepo(helmConfig)
		args = append(args, chartName)
		if chartRepo != "" {
			args = append(args, "--repo", chartRepo)
			args = append(args, "--repository-config=''")
		}
		if helmConfig.Chart.Version != "" {
			args = append(args, "--version", helmConfig.Chart.Version)
		}
		if helmConfig.Chart.Username != "" {
			args = append(args, "--username", helmConfig.Chart.Username)
		}
		if helmConfig.Chart.Password != "" {
			args = append(args, "--password", helmConfig.Chart.Password)
		}
	}

	// Upgrade options
	args = append(args, helmConfig.UpgradeArgs...)
	output, err := c.genericHelm.Exec(ctx, args)
	if helmConfig.DisplayOutput {
		writer := ctx.Log.Writer(logrus.InfoLevel)
		_, _ = writer.Write(output)
		_ = writer.Close()
	}
	if err != nil {
		return nil, err
	}

	releases, err := c.ListReleases(ctx, releaseNamespace)
	if err != nil {
		return nil, err
	}

	for _, r := range releases {
		if r.Name == releaseName && r.Namespace == releaseNamespace {
			return r, nil
		}
	}

	return nil, nil
}

func (c *client) Template(ctx *devspacecontext.Context, releaseName, releaseNamespace string, values map[string]interface{}, helmConfig *latest.HelmConfig) (string, error) {
	cleanup, chartDir, err := c.genericHelm.FetchChart(ctx, helmConfig)
	if err != nil {
		return "", err
	} else if cleanup {
		defer os.RemoveAll(filepath.Dir(chartDir))
	}

	if releaseNamespace == "" {
		releaseNamespace = ctx.KubeClient.Namespace()
	}

	valuesFile, err := c.genericHelm.WriteValues(values)
	if err != nil {
		return "", err
	}
	defer os.Remove(valuesFile)

	args := []string{
		"template",
		releaseName,
		chartDir,
		"--namespace",
		releaseNamespace,
		"--values",
		valuesFile,
	}
	args = append(args, helmConfig.TemplateArgs...)
	result, err := c.genericHelm.Exec(ctx, args)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func (c *client) DeleteRelease(ctx *devspacecontext.Context, releaseName string, releaseNamespace string) error {
	if releaseNamespace == "" {
		releaseNamespace = ctx.KubeClient.Namespace()
	}

	args := []string{
		"delete",
		releaseName,
		"--namespace",
		releaseNamespace,
	}
	_, err := c.genericHelm.Exec(ctx, args)
	if err != nil {
		return err
	}

	return nil
}

func (c *client) ListReleases(ctx *devspacecontext.Context, namespace string) ([]*types.Release, error) {
	args := []string{
		"list",
		"--namespace",
		namespace,
		"--max",
		strconv.Itoa(0),
		"--output",
		"json",
	}
	out, err := c.genericHelm.Exec(ctx, args)
	if err != nil {
		return nil, err
	}

	releases := []*types.Release{}
	err = yaml.Unmarshal(out, &releases)
	if err != nil {
		return nil, err
	}

	return releases, nil
}
