package remove

import (
	"errors"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/spf13/cobra"
)

type syncCmd struct {
	*flags.GlobalFlags

	LabelSelector string
	LocalPath     string
	ContainerPath string
	RemoveAll     bool
}

func newSyncCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
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
		RunE: cmd.RunRemoveSync,
	}

	syncCmd.Flags().StringVar(&cmd.LabelSelector, "label-selector", "", "Comma separated key=value selector list (e.g. release=test)")
	syncCmd.Flags().StringVar(&cmd.LocalPath, "local", "", "Relative local path to remove")
	syncCmd.Flags().StringVar(&cmd.ContainerPath, "container", "", "Absolute container path to remove")
	syncCmd.Flags().BoolVar(&cmd.RemoveAll, "all", false, "Remove all configured sync paths")

	return syncCmd
}

// RunRemoveSync executes the remove sync command logic
func (cmd *syncCmd) RunRemoveSync(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configLoader := loader.NewConfigLoader(cmd.ToConfigOptions(), log.GetInstance())
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	config, err := configLoader.LoadWithoutProfile()
	if err != nil {
		return err
	}

	err = configure.RemoveSyncPath(config, cmd.RemoveAll, cmd.LocalPath, cmd.ContainerPath, cmd.LabelSelector)
	if err != nil {
		return err
	}

	return configLoader.Save(config)
}
