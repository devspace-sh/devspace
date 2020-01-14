package add

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/message"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type syncCmd struct {
	*flags.GlobalFlags

	LabelSelector string
	LocalPath     string
	ContainerPath string
	ExcludedPaths string
}

func newSyncCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &syncCmd{GlobalFlags: globalFlags}

	addSyncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Add a sync path",
		Long: `
#######################################################
################# devspace add sync ###################
#######################################################
Add a sync path to this project's devspace.yaml

Example:
devspace add sync --local=app --container=/app
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunAddSync(f, cobraCmd, args)
		},
	}

	addSyncCmd.Flags().StringVar(&cmd.LabelSelector, "label-selector", "", "Comma separated key=value selector list (e.g. release=test)")
	addSyncCmd.Flags().StringVar(&cmd.LocalPath, "local", "", "Relative local path")
	addSyncCmd.Flags().StringVar(&cmd.ContainerPath, "container", "", "Absolute container path")
	addSyncCmd.Flags().StringVar(&cmd.ExcludedPaths, "exclude", "", "Comma separated list of paths to exclude (e.g. node_modules/,bin,*.exe)")

	addSyncCmd.MarkFlagRequired("local")
	addSyncCmd.MarkFlagRequired("container")

	return addSyncCmd
}

// RunAddSync executes the add sync command logic
func (cmd *syncCmd) RunAddSync(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	logger := f.GetLog()
	configLoader := f.NewConfigLoader(cmd.ToConfigOptions(), logger)
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

	err = configure.AddSyncPath(config, cmd.LocalPath, cmd.ContainerPath, cmd.Namespace, cmd.LabelSelector, cmd.ExcludedPaths)
	if err != nil {
		return errors.Wrap(err, "add sync path")
	}

	err = configLoader.Save(config)
	if err != nil {
		return err
	}

	logger.Donef("Successfully added sync between local path %v and container path %v", cmd.LocalPath, cmd.ContainerPath)
	return nil
}
