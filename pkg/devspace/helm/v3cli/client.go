package v3cli

import (
	"runtime"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/abstractcli"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"gopkg.in/yaml.v2"
)

var (
	helmVersionPrefix = "v3"
	helmVersion       = "v3.3.4"
	helmDownload      = "https://get.helm.sh/helm-" + helmVersion + "-" + runtime.GOOS + "-amd64"
)

type client struct {
	client abstractcli.Client
}

// NewClient creates a new helm client
func NewClient(config *latest.Config, kubeClient kubectl.Client, log log.Logger) (types.Client, error) {
	c := &client{}

	abstractClient, err := abstractcli.NewClient(config, kubeClient, "", helmVersionPrefix, helmDownload, c.parseReleaseOutput, log)
	if err != nil {
		return nil, err
	}

	c.client = abstractClient
	return c, nil
}

func (c *client) InstallChart(releaseName string, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (*types.Release, error) {
	return c.client.InstallChart(releaseName, releaseNamespace, values, helmConfig)
}

func (c *client) Template(releaseName, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (string, error) {
	getArgs := func(chartDir, releaseNamespace, file, context, tillerNamespace string) []string {
		return []string{"template", releaseName, chartDir, "--namespace", releaseNamespace, "--values", file, "--kube-context", context}
	}
	return c.client.Template(releaseNamespace, values, helmConfig, "pull", getArgs)
}

func (c *client) DeleteRelease(releaseName string, releaseNamespace string, helmConfig *latest.HelmConfig) error {
	return c.client.DeleteRelease(releaseName, releaseNamespace, helmConfig, "uninstall", []string{})
}

func (c *client) ListReleases(helmConfig *latest.HelmConfig) ([]*types.Release, error) {
	return c.client.ListReleases(helmConfig)
}

func (c *client) parseReleaseOutput(out []byte) ([]*types.Release, error) {
	releases := &[]*types.Release{}
	err := yaml.Unmarshal(out, releases)
	if err != nil {
		return nil, err
	}

	return *releases, nil
}
