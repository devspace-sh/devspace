package add

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type portCmd struct {
	*flags.GlobalFlags

	LabelSelector string
}

func newPortCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &portCmd{GlobalFlags: globalFlags}

	addPortCmd := &cobra.Command{
		Use:   "port",
		Short: "Add a new port forward configuration",
		Long: `
#######################################################
################ devspace add port ####################
#######################################################
Add a new port mapping to this project's devspace.yaml

Format is port(:remotePort) comma separated, e.g.
devspace add port 8080:80,3000
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunAddPort(f, cobraCmd, args)
		}}

	addPortCmd.Flags().StringVar(&cmd.LabelSelector, "label-selector", "", "Comma separated key=value label-selector list (e.g. release=test)")

	return addPortCmd
}

// RunAddPort executes the add port command logic
func (cmd *portCmd) RunAddPort(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	logger := f.GetLog()
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(logger)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	logger.Warn("This command is deprecated and will be removed in a future DevSpace version. Please modify the devspace.yaml directly instead")
	configInterface, err := configLoader.Load(cmd.ToConfigOptions(), logger)
	if err != nil {
		return err
	}

	config := configInterface.Config()
	configureManager := f.NewConfigureManager(config, logger)

	err = configureManager.AddPort(cmd.Namespace, cmd.LabelSelector, args)
	if err != nil {
		return err
	}

	err = configLoader.Save(config)
	if err != nil {
		return err
	}

	logger.Donef("Successfully added port %v", args[0])
	return nil
}
