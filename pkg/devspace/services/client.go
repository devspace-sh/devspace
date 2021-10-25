package services

import (
	"io"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	dependencytypes "github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/services/podreplace"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// Client implements all service functions
type Client interface {
	StartAttach(options targetselector.Options, interrupt chan error) error

	StartLogs(options targetselector.Options, follow bool, tail int64, wait bool) error
	StartLogsWithWriter(options targetselector.Options, follow bool, tail int64, wait bool, writer io.Writer) error

	StartPortForwarding(interrupt chan error, prefixFn PrefixFn) error
	StartSync(interrupt chan error, printSyncLog bool, verboseSync bool, prefixFn PrefixFn) error

	StartSyncFromCmd(options targetselector.Options, syncConfig *latest.SyncConfig, interrupt chan error, noWatch, verbose bool) error
	StartTerminal(options targetselector.Options, args []string, workDir string, interrupt chan error, wait, restart bool, subcommand string, stdout io.Writer, stderr io.Writer, stdin io.Reader) (int, error)

	ReplacePods(prefixFn PrefixFn) error

	Log() log.Logger
	KubeClient() kubectl.Client
	Config() config.Config
	Dependencies() []dependencytypes.Dependency
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

func (serviceClient *client) Config() config.Config {
	return serviceClient.config
}
func (serviceClient *client) Log() log.Logger {
	return serviceClient.log
}
func (serviceClient *client) KubeClient() kubectl.Client {
	return serviceClient.client
}
func (serviceClient *client) Dependencies() []dependencytypes.Dependency {
	return serviceClient.dependencies
}
