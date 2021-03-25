package v2

import (
	"github.com/loft-sh/devspace/pkg/devspace/helm/generic"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/helm/types"
	"gopkg.in/yaml.v2"
)

// InstallChart installs the given chart via helm v2
func (c *client) InstallChart(releaseName string, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (*types.Release, error) {
	c.log.Warn("Helm v2 support is deprecated and will be removed in future (see https://helm.sh/blog/helm-v2-deprecation-timeline/) for more details.")
	err := c.ensureTiller(helmConfig)
	if err != nil {
		return nil, err
	}

	valuesFile, err := c.genericHelm.WriteValues(values)
	if err != nil {
		return nil, err
	}
	defer os.Remove(valuesFile)

	if releaseNamespace == "" {
		releaseNamespace = c.kubeClient.Namespace()
	}

	chartName, chartRepo := generic.ChartNameAndRepo(helmConfig)
	args := []string{
		"upgrade",
		releaseName,
		chartName,
		"--namespace",
		releaseNamespace,
		"--values",
		valuesFile,
		"--install",
		"--tiller-namespace",
		c.tillerNamespace,
	}

	// Chart settings
	if chartRepo != "" {
		args = append(args, "--repo", chartRepo)
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

	// Upgrade options
	if helmConfig.Atomic || helmConfig.CleanupOnFail {
		args = append(args, "--cleanup-on-fail")
	}
	if helmConfig.Wait {
		args = append(args, "--wait")
	}
	if helmConfig.Timeout != nil {
		args = append(args, "--timeout", strconv.FormatInt(*helmConfig.Timeout, 10))
	}
	if helmConfig.Recreate {
		args = append(args, "--recreate-pods")
	}
	if helmConfig.Force {
		args = append(args, "--force")
	}
	if helmConfig.DisableHooks {
		args = append(args, "--no-hooks")
	}

	args = append(args, helmConfig.UpgradeArgs...)
	for {
		_, err = c.genericHelm.Exec(args, helmConfig)
		if err != nil {
			if strings.Index(err.Error(), "could not find a ready tiller pod") != -1 {
				time.Sleep(time.Second * 3)
				err = c.ensureTiller(helmConfig)
				if err != nil {
					return nil, err
				}

				continue
			}

			return nil, err
		}

		break
	}

	releases, err := c.ListReleases(helmConfig)
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

func (c *client) Template(releaseName, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (string, error) {
	cleanup, chartDir, err := c.genericHelm.FetchChart(helmConfig)
	if err != nil {
		return "", err
	} else if cleanup {
		defer os.RemoveAll(filepath.Dir(chartDir))
	}

	if releaseNamespace == "" {
		releaseNamespace = c.kubeClient.Namespace()
	}

	valuesFile, err := c.genericHelm.WriteValues(values)
	if err != nil {
		return "", err
	}
	defer os.Remove(valuesFile)

	args := []string{
		"template",
		chartDir,
		"--name",
		releaseName,
		"--namespace",
		releaseNamespace,
		"--values",
		valuesFile,
		"--tiller-namespace",
		c.tillerNamespace,
	}
	args = append(args, helmConfig.TemplateArgs...)
	result, err := c.genericHelm.Exec(args, helmConfig)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func (c *client) DeleteRelease(releaseName string, releaseNamespace string, helmConfig *latest.HelmConfig) error {
	err := c.ensureTiller(helmConfig)
	if err != nil {
		return err
	}

	args := []string{
		"delete",
		releaseName,
		"--tiller-namespace",
		c.tillerNamespace,
		"--purge",
	}
	args = append(args, helmConfig.DeleteArgs...)
	_, err = c.genericHelm.Exec(args, helmConfig)
	if err != nil {
		return err
	}

	return nil
}

func (c *client) ListReleases(helmConfig *latest.HelmConfig) ([]*types.Release, error) {
	err := c.ensureTiller(helmConfig)
	if err != nil {
		return nil, err
	}

	args := []string{
		"list",
		"--tiller-namespace",
		c.tillerNamespace,
		"--output",
		"json",
	}
	out, err := c.genericHelm.Exec(args, helmConfig)
	if err != nil {
		if strings.Index(string(out), "could not find a ready tiller pod") > -1 {
			c.log.Info("Couldn't find a ready tiller pod, will wait 3 seconds more")
			time.Sleep(time.Second * 3)
			return c.ListReleases(helmConfig)
		}

		return nil, err
	}

	releases := &struct {
		Releases []struct {
			Name      string `yaml:"Name"`
			Namespace string `yaml:"Namespace"`
			Status    string `yaml:"Status"`
			Revision  int32  `yaml:"Revision"`
			Updated   string `yaml:"Updated"`
		} `yaml:"Releases"`
	}{}
	err = yaml.Unmarshal(out, releases)
	if err != nil {
		return nil, err
	}

	result := []*types.Release{}
	for _, release := range releases.Releases {
		result = append(result, &types.Release{
			Name:         release.Name,
			Namespace:    release.Namespace,
			Status:       release.Status,
			Revision:     strconv.Itoa(int(release.Revision)),
			LastDeployed: release.Updated,
		})
	}

	return result, nil
}
