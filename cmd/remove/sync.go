package remove

import (
	"errors"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/spf13/cobra"
)

type syncCmd struct {
	*flags.GlobalFlags

	LabelSelector string
	LocalPath     string
	ContainerPath string
	RemoveAll     bool
}

func newSyncCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &syncCmd{GlobalFlags: globalFlags}

	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Remove sync paths from the devspace",
		Long: `
#######################################################
############### devspace remove sync ##################
#######################################################
Remove sync paths from the devspace

How to use:
devspace remove sync --local=app
devspace remove sync --container=/app
devspace remove sync --label-selector=release=test
devspace remove sync --all
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunRemoveSync(f, cobraCmd, args)
		}}

	syncCmd.Flags().StringVar(&cmd.LabelSelector, "label-selector", "", "Comma separated key=value selector list (e.g. release=test)")
	syncCmd.Flags().StringVar(&cmd.LocalPath, "local", "", "Relative local path to remove")
	syncCmd.Flags().StringVar(&cmd.ContainerPath, "container", "", "Absolute container path to remove")
	syncCmd.Flags().BoolVar(&cmd.RemoveAll, "all", false, "Remove all configured sync paths")

	return syncCmd
}

// RunRemoveSync executes the remove sync command logic
func (cmd *syncCmd) RunRemoveSync(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	log.Warn("This command is deprecated and will be removed in a future DevSpace version. Please modify the devspace.yaml directly instead")
	config, err := configLoader.Load(cmd.ToConfigOptions(), log)
	if err != nil {
		return err
	}

	configureManager := f.NewConfigureManager(config.Config(), log)
	err = configureManager.RemoveSyncPath(cmd.RemoveAll, cmd.LocalPath, cmd.ContainerPath, cmd.LabelSelector)
	if err != nil {
		return err
	}

	return configLoader.Save(config.Config())
}
