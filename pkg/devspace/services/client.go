package services

import (
	"github.com/loft-sh/devspace/pkg/devspace/config"
	dependencytypes "github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/services/podreplace"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"io"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// Client implements all service functions
type Client interface {
	StartAttach(options targetselector.Options, interrupt chan error) error

	StartLogs(options targetselector.Options, follow bool, tail int64, wait bool) error
	StartLogsWithWriter(options targetselector.Options, follow bool, tail int64, wait bool, writer io.Writer) error

	StartPortForwarding(interrupt chan error) error
	StartReversePortForwarding(interrupt chan error) error
	StartSync(interrupt chan error, printSyncLog bool, verboseSync bool, prefixFn func(idx int, syncConfig *latest.SyncConfig) string) error

	StartSyncFromCmd(options targetselector.Options, syncConfig *latest.SyncConfig, interrupt chan error, verbose bool) error
	StartTerminal(options targetselector.Options, args []string, workDir string, interrupt chan error, wait bool) (int, error)

	ReplacePods() error
}

type client struct {
	config       config.Config
	dependencies []dependencytypes.Dependency

	podReplacer podreplace.PodReplacer
	client      kubectl.Client
	log         log.Logger
}

// NewClient creates a new client object
func NewClient(config config.Config, dependencies []dependencytypes.Dependency, kubeClient kubectl.Client, log log.Logger) Client {
	return &client{
		config:       config,
		dependencies: dependencies,
		client:       kubeClient,
		podReplacer:  podreplace.NewPodReplacer(),
		log:          log,
	}
}
