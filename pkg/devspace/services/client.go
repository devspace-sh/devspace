package services

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"io"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// Client implements all service functions
type Client interface {
	StartAttach(options targetselector.Options, interrupt chan error) error

	StartLogs(options targetselector.Options, follow bool, tail int64) error
	StartLogsWithWriter(options targetselector.Options, follow bool, tail int64, writer io.Writer) error

	StartPortForwarding(interrupt chan error) error
	StartReversePortForwarding(interrupt chan error) error
	StartSync(interrupt chan error, printSyncLog bool, verboseSync bool) error

	StartSyncFromCmd(options targetselector.Options, syncConfig *latest.SyncConfig, interrupt chan error, verbose bool) error
	StartTerminal(options targetselector.Options, args []string, interrupt chan error, wait bool) (int, error)
}

type client struct {
	config    *latest.Config
	generated *generated.Config
	client    kubectl.Client
	log       log.Logger
}

// NewClient creates a new client object
func NewClient(config *latest.Config, generated *generated.Config, kubeClient kubectl.Client, log log.Logger) Client {
	return &client{
		config:    config,
		generated: generated,
		client:    kubeClient,
		log:       log,
	}
}
