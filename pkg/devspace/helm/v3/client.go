package v3

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	dependencyutil "github.com/loft-sh/devspace/pkg/devspace/dependency/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/helm/generic"
	"github.com/loft-sh/devspace/pkg/devspace/helm/types"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/loft-util/pkg/downloader/commands"
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

func (c *client) DownloadChart(ctx devspacecontext.Context, helmConfig *latest.HelmConfig) (string, error) {
	chartName, err := dependencyutil.DownloadDependency(ctx.Context(), ctx.WorkingDir(), helmConfig.Chart.Source, ctx.Log())
	if err != nil {
		return "", err
	}
	return filepath.Dir(chartName), nil
}

// InstallChart installs the given chart via helm v3
func (c *client) InstallChart(ctx devspacecontext.Context, releaseName string, releaseNamespace string, values map[string]interface{}, helmConfig *latest.HelmConfig) (*types.Release, error) {
	valuesFile, err := c.genericHelm.WriteValues(values)
	if err != nil {
		return nil, err
	}
	defer os.Remove(valuesFile)

	if releaseNamespace == "" {
		releaseNamespace = ctx.KubeClient().Namespace()
	}

	args := []string{
		"upgrade",
		releaseName,
		"--values",
		valuesFile,
		"--install",
	}
	if releaseNamespace != "" {
		args = append(args, "--namespace", releaseNamespace)
	}

	// Chart settings
	chartPath := ""
	if helmConfig.Chart.Source != nil {
		dependencyPath, err := dependencyutil.GetDependencyPath(ctx.WorkingDir(), helmConfig.Chart.Source)
		if err != nil {
			return nil, err
		}

		chartPath = filepath.Dir(dependencyPath)
		args = append(args, chartPath)
	} else {
		chartName, chartRepo := generic.ChartNameAndRepo(helmConfig)
		chartPath = filepath.Join(ctx.WorkingDir(), chartName)
		args = append(args, chartName)
		if chartRepo != "" {
			args = append(args, "--repo", chartRepo)
			args = append(args, "--repository-config=''")
		}
		if helmConfig.Chart.Version != "" {
			args = append(args, "--version", helmConfig.Chart.Version)
		}

		// log into OCI registry if specified
		if strings.HasPrefix(chartName, "oci://") {
			if helmConfig.Chart.Username != "" && helmConfig.Chart.Password != "" {
				chartNameURL, err := url.Parse(chartName)
				if err != nil {
					return nil, errors.Wrap(err, "chartName malformed for oci registry")
				}

				_, err = c.genericHelm.Exec(ctx, []string{"registry", "login", chartNameURL.Hostname(), "--username", helmConfig.Chart.Username, "--password", helmConfig.Chart.Password})
				if err != nil {
					return nil, errors.Wrap(err, "login oci registry")
				}
			}
		} else {
			if helmConfig.Chart.Username != "" {
				args = append(args, "--username", helmConfig.Chart.Username)
			}
			if helmConfig.Chart.Password != "" {
				args = append(args, "--password", helmConfig.Chart.Password)
			}
		}
	}

	// Update dependencies if needed
	if helmConfig.DisableDependencyUpdate == nil || (helmConfig.DisableDependencyUpdate != nil && !*helmConfig.DisableDependencyUpdate) {
		stat, err := os.Stat(chartPath)
		if err == nil && stat.IsDir() {
			args = append(args, "--dependency-update")
		}
	}
	// Upgrade options
	args = append(args, helmConfig.UpgradeArgs...)
	output, err := c.genericHelm.Exec(ctx, args)
	if helmConfig.DisplayOutput {
		writer := ctx.Log().Writer(logrus.InfoLevel, false)
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

func (c *client) Template(ctx devspacecontext.Context, releaseName, releaseNamespace string, values map[string]interface{}, helmConfig *latest.HelmConfig) (string, error) {
	valuesFile, err := c.genericHelm.WriteValues(values)
	if err != nil {
		return "", err
	}
	defer os.Remove(valuesFile)

	if releaseNamespace == "" {
		releaseNamespace = ctx.KubeClient().Namespace()
	}

	args := []string{
		"template",
		releaseName,
		"--values",
		valuesFile,
	}
	if releaseNamespace != "" {
		args = append(args, "--namespace", releaseNamespace)
	}

	// Chart settings
	chartPath := ""
	if helmConfig.Chart.Source != nil {
		dependencyPath, err := dependencyutil.GetDependencyPath(ctx.WorkingDir(), helmConfig.Chart.Source)
		if err != nil {
			return "", err
		}

		chartPath = filepath.Dir(dependencyPath)
		args = append(args, chartPath)
	} else {
		chartName, chartRepo := generic.ChartNameAndRepo(helmConfig)
		chartPath = filepath.Join(ctx.WorkingDir(), chartName)
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

	// Update dependencies if needed
	if helmConfig.DisableDependencyUpdate == nil || (helmConfig.DisableDependencyUpdate != nil && !*helmConfig.DisableDependencyUpdate) {
		stat, err := os.Stat(chartPath)
		if err == nil && stat.IsDir() {
			args = append(args, "--dependency-update")
		}
	}
	args = append(args, helmConfig.TemplateArgs...)
	result, err := c.genericHelm.Exec(ctx, args)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func (c *client) DeleteRelease(ctx devspacecontext.Context, releaseName string, releaseNamespace string) error {
	if releaseNamespace == "" {
		releaseNamespace = ctx.KubeClient().Namespace()
	}

	args := []string{
		"delete",
		releaseName,
	}
	if releaseNamespace != "" {
		args = append(args, "--namespace", releaseNamespace)
	}
	_, err := c.genericHelm.Exec(ctx, args)
	if err != nil {
		return err
	}

	return nil
}

func (c *client) ListReleases(ctx devspacecontext.Context, namespace string) ([]*types.Release, error) {
	args := []string{
		"list",
		"--max",
		strconv.Itoa(0),
		"--output",
		"json",
	}
	if namespace != "" {
		args = append(args, "--namespace", namespace)
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
