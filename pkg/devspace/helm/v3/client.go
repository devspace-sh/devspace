package v3

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/helm/generic"
	"github.com/loft-sh/devspace/pkg/devspace/helm/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/loft-sh/devspace/pkg/util/log"

	"runtime"
	"strings"
)

var (
	helmVersion  = "v3.6.2"
	helmDownload = "https://get.helm.sh/helm-" + helmVersion + "-" + runtime.GOOS + "-" + runtime.GOARCH
)

type Client struct {
	exec        command.Exec
	kubeClient  kubectl.Client
	genericHelm generic.Client

	log log.Logger
}

// NewClient creates a new helm v3 Client
func NewClient(kubeClient kubectl.Client, log log.Logger) (types.Client, error) {
	c := &Client{
		exec:       command.NewStreamCommand,
		kubeClient: kubeClient,
		log:        log,
	}

	c.genericHelm = generic.NewGenericClient(c, log)
	return c, nil
}

func (c *Client) IsInCluster() bool {
	return c.kubeClient.IsInCluster()
}

func (c *Client) KubeContext() string {
	return c.kubeClient.CurrentContext()
}

func (c *Client) Command() string {
	return "helm"
}

func (c *Client) DownloadURL() string {
	return helmDownload
}

func (c *Client) IsValidHelm(path string) (bool, error) {
	out, err := c.exec(path, []string{"version"}).Output()
	if err != nil {
		return false, nil
	}

	return strings.Contains(string(out), `:"v3.`), nil
}

// InstallChart installs the given chart via helm v3
func (c *Client) InstallChart(releaseName string, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (*types.Release, error) {
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
	}

	// Chart settings
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

	// Upgrade options
	if helmConfig.Atomic {
		args = append(args, "--atomic")
	}
	if helmConfig.CleanupOnFail {
		args = append(args, "--cleanup-on-fail")
	}
	if helmConfig.Wait {
		args = append(args, "--wait")
	}
	if helmConfig.Timeout != nil {
		args = append(args, "--timeout", strconv.FormatInt(*helmConfig.Timeout, 10))
	}
	if helmConfig.Force {
		args = append(args, "--force")
	}
	if helmConfig.DisableHooks {
		args = append(args, "--no-hooks")
	}

	args = append(args, helmConfig.UpgradeArgs...)
	output, err := c.genericHelm.Exec(args, helmConfig)

	if helmConfig.DisplayOutput {
		_, _ = c.log.Write(output)
	}

	if err != nil {
		return nil, err
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

func (c *Client) Template(releaseName, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (string, error) {
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
		releaseName,
		chartDir,
		"--namespace",
		releaseNamespace,
		"--values",
		valuesFile,
	}
	args = append(args, helmConfig.TemplateArgs...)
	result, err := c.genericHelm.Exec(args, helmConfig)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func (c *Client) DeleteRelease(releaseName string, releaseNamespace string, helmConfig *latest.HelmConfig) error {
	if releaseNamespace == "" {
		releaseNamespace = c.kubeClient.Namespace()
	}

	args := []string{
		"delete",
		releaseName,
		"--namespace",
		releaseNamespace,
	}
	args = append(args, helmConfig.DeleteArgs...)
	_, err := c.genericHelm.Exec(args, helmConfig)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) ListReleases(helmConfig *latest.HelmConfig) ([]*types.Release, error) {
	args := []string{
		"list",
		"--namespace",
		c.kubeClient.Namespace(),
		"--output",
		"json",
	}
	out, err := c.genericHelm.Exec(args, helmConfig)
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
