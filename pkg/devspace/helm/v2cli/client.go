package v2cli

import (
	"runtime"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/helm/abstractcli"
	"gopkg.in/yaml.v2"

	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
)

var (
	helmVersionPrefix = "v2"
	helmVersion       = "v2.16.9"
	helmDownload      = "https://get.helm.sh/helm-" + helmVersion + "-" + runtime.GOOS + "-amd64"
)

type client struct {
	client abstractcli.Client

	log log.Logger
}

// NewClient creates a new helm client
func NewClient(config *latest.Config, kubeClient kubectl.Client, tillerNamespace string, log log.Logger) (types.Client, error) {
	if tillerNamespace == "" {
		tillerNamespace = kubeClient.Namespace()
	}

	c := &client{log: log}

	abstractClient, err := abstractcli.NewClient(config, kubeClient, tillerNamespace, helmVersionPrefix, helmDownload, c.parseReleaseOutput, log)
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
		return []string{"template", chartDir, "--name", releaseName, "--namespace", releaseNamespace, "--values", file, "--kube-context", context, "--tiller-namespace", tillerNamespace}
	}
	return c.client.Template(releaseNamespace, values, helmConfig, "fetch", getArgs)
}

func (c *client) DeleteRelease(releaseName string, releaseNamespace string, helmConfig *latest.HelmConfig) error {
	return c.client.DeleteRelease(releaseName, releaseNamespace, helmConfig, "delete", []string{"--purge"})
}

func (c *client) ListReleases(helmConfig *latest.HelmConfig) ([]*types.Release, error) {
	return c.client.ListReleases(helmConfig)
}

func (c *client) parseReleaseOutput(out []byte) ([]*types.Release, error) {
	releases := &struct {
		Releases []struct {
			Name      string `yaml:"Name"`
			Namespace string `yaml:"Namespace"`
			Status    string `yaml:"Status"`
			Revision  int32  `yaml:"Revision"`
			Updated   string `yaml:"Updated"`
		} `yaml:"Releases"`
	}{}
	err := yaml.Unmarshal(out, releases)
	if err != nil {
		return nil, err
	}

	result := []*types.Release{}
	for _, release := range releases.Releases {
		t, err := time.ParseInLocation(time.ANSIC, release.Updated, time.Local)
		if err != nil {
			return nil, err
		}

		result = append(result, &types.Release{
			Name:         release.Name,
			Namespace:    release.Namespace,
			Status:       release.Status,
			Version:      release.Revision,
			LastDeployed: t,
		})
	}

	return result, nil
}
