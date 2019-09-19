package cmd

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
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
	syncCmd.Flags().BoolVar(&cmd.Verbose, "verbose", false, "Shows every file that is synced")

	return syncCmd
}

// Run executes the command logic
func (cmd *SyncCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// Load generated config if possible
	var err error
	var generatedConfig *generated.Config
	if configutil.ConfigExists() {
		generatedConfig, err = generated.LoadConfig(cmd.Profile)
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
	err = cloud.ResumeSpace(client, true, log.GetInstance())
	if err != nil {
		return err
	}

	var config *latest.Config
	if configutil.ConfigExists() {
		config, err = configutil.GetConfig(cmd.ToConfigOptions())
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

	// Start terminal
	err = services.StartSyncFromCmd(config, client, params, cmd.LocalPath, cmd.ContainerPath, cmd.Exclude, cmd.Verbose, log.GetInstance())
	if err != nil {
		return err
	}

	return nil
}
