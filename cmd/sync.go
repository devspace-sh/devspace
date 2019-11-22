package cmd

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/resume"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	latest "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// SyncCmd is a struct that defines a command call for "sync"
type SyncCmd struct {
	*flags.GlobalFlags

	LabelSelector string
	Container     string
	Pod           string
	Pick          bool

	Exclude       []string
	ContainerPath string
	LocalPath     string
	Verbose       bool

	NoWatch               bool
	DownloadOnInitialSync bool
}

// NewSyncCmd creates a new init command
func NewSyncCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &SyncCmd{GlobalFlags: globalFlags}

	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Starts a bi-directional sync between the target container and the local path",
		Long: `
#######################################################
################### devspace sync #####################
#######################################################
Starts a bi-directionaly sync between the target container
and the current path:

devspace sync
devspace sync --local-path=subfolder --container-path=/app
devspace sync --exclude=node_modules --exclude=test
devspace sync --pod=my-pod --container=my-container
devspace sync --container-path=/my-path
#######################################################`,
		RunE: cmd.Run,
	}

	syncCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	syncCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to open a shell to")
	syncCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	syncCmd.Flags().BoolVar(&cmd.Pick, "pick", false, "Select a pod")

	syncCmd.Flags().StringSliceVarP(&cmd.Exclude, "exclude", "e", []string{}, "Exclude directory from sync")
	syncCmd.Flags().StringVar(&cmd.LocalPath, "local-path", ".", "Local path to use (Default is current directory")
	syncCmd.Flags().StringVar(&cmd.ContainerPath, "container-path", "", "Container path to use (Default is working directory)")
	syncCmd.Flags().BoolVar(&cmd.DownloadOnInitialSync, "download-on-initial-sync", true, "Downloads all locally non existing remote files in the beginning")
	syncCmd.Flags().BoolVar(&cmd.NoWatch, "no-watch", false, "Synchronizes local and remote and then stops")
	syncCmd.Flags().BoolVar(&cmd.Verbose, "verbose", false, "Shows every file that is synced")

	return syncCmd
}

// Run executes the command logic
func (cmd *SyncCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// Load generated config if possible
	var err error
	var generatedConfig *generated.Config

	configLoader := loader.NewConfigLoader(cmd.ToConfigOptions(), log.GetInstance())
	if configLoader.Exists() {
		generatedConfig, err = configLoader.Generated()
		if err != nil {
			return err
		}
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, log.GetInstance())
	if err != nil {
		return err
	}

	// Get config with adjusted cluster config
	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Wrap(err, "new kube client")
	}

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, log.GetInstance())
	if err != nil {
		return err
	}

	// Signal that we are working on the space if there is any
	err = resume.NewSpaceResumer(client, log.GetInstance()).ResumeSpace(true)
	if err != nil {
		return err
	}

	var config *latest.Config
	if configLoader.Exists() {
		config, err = configLoader.Load()
		if err != nil {
			return err
		}
	}

	// Build params
	params := targetselector.CmdParameter{
		ContainerName: cmd.Container,
		LabelSelector: cmd.LabelSelector,
		Namespace:     cmd.Namespace,
		PodName:       cmd.Pod,
	}
	if cmd.Pick != false {
		params.Pick = &cmd.Pick
	}

	selectorParameter := &targetselector.SelectorParameter{
		CmdParameter: params,
	}

	// Start terminal
	servicesClient := services.NewClient(config, generatedConfig, client, selectorParameter, log.GetInstance())
	err = servicesClient.StartSyncFromCmd(cmd.LocalPath, cmd.ContainerPath, cmd.Exclude, cmd.Verbose, cmd.DownloadOnInitialSync, cmd.NoWatch)
	if err != nil {
		return err
	}

	return nil
}
