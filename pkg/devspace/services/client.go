package services

import (
	"io"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// Client implements all service functions
type Client interface {
	StartAttach(imageSelector []string, interrupt chan error) error

	StartLogs(follow bool, tail int64) error
	StartLogsWithWriter(follow bool, tail int64, writer io.Writer) error

	StartPortForwarding() error

	StartSyncFromCmd(localPath, containerPath string, exclude []string, verbose, downloadOnInitialSync, waitInitialSync bool) error
	StartSync(verboseSync bool) error

	StartTerminal(args []string, imageSelector []string, interrupt chan error, wait bool) (int, error)
}

type client struct {
	config    *latest.Config
	generated *generated.Config
	client    kubectl.Client

	log log.Logger

	selectorParameter *targetselector.SelectorParameter
}

// NewClient creates a new client object
func NewClient(config *latest.Config, generated *generated.Config, kubeClient kubectl.Client, selectorParameter *targetselector.SelectorParameter, log log.Logger) Client {
	return &client{
		config:            config,
		generated:         generated,
		client:            kubeClient,
		log:               log,
		selectorParameter: selectorParameter,
	}
}
