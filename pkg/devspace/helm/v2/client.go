package v2

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/generic"
	"runtime"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/command"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

var (
	helmVersion  = "v2.17.0"
	helmDownload = "https://get.helm.sh/helm-" + helmVersion + "-" + runtime.GOOS + "-amd64"
)

type client struct {
	config *latest.Config

	exec            command.Exec
	kubeClient      kubectl.Client
	tillerNamespace string
	genericHelm     generic.Client

	log log.Logger
}

// NewClient creates a new helm client
func NewClient(config *latest.Config, kubeClient kubectl.Client, tillerNamespace string, log log.Logger) (types.Client, error) {
	if tillerNamespace == "" {
		tillerNamespace = kubeClient.Namespace()
	}

	c := &client{
		config: config,

		exec:            command.NewStreamCommand,
		kubeClient:      kubeClient,
		tillerNamespace: tillerNamespace,

		log: log,
	}
	c.genericHelm = generic.NewGenericClient(c, log)
	return c, nil
}

func (c *client) IsInCluster() bool {
	return c.kubeClient.IsInCluster()
}

func (c *client) KubeContext() string {
	return c.kubeClient.CurrentContext()
}

func (c *client) Command() string {
	return "helm2"
}

func (c *client) DownloadURL() string {
	return helmDownload
}

func (c *client) IsValidHelm(path string) (bool, error) {
	out, err := c.exec(path, []string{"version", "--client"}).Output()
	if err != nil {
		return false, nil
	}

	return strings.Contains(string(out), `:"v2.`), nil
}
