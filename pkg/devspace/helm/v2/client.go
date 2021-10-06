package v2

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/helm/generic"
	"github.com/loft-sh/devspace/pkg/devspace/helm/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/loft-sh/devspace/pkg/util/downloader/commands"
	"github.com/loft-sh/devspace/pkg/util/log"
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

func (c *client) Command() commands.Command {
	return commands.NewHelmV2Command()
}
