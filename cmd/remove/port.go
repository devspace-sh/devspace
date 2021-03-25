package remove

import (
	"errors"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/spf13/cobra"
)

type portCmd struct {
	*flags.GlobalFlags

	LabelSelector string
	RemoveAll     bool
}

func newPortCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &portCmd{GlobalFlags: globalFlags}

	portCmd := &cobra.Command{
		Use:   "port",
		Short: "Removes forwarded ports from a devspace",
		Long: `
#######################################################
############### devspace remove port ##################
#######################################################
Removes port mappings from the devspace configuration:
devspace remove port 8080,3000
devspace remove port --label-selector=release=test
devspace remove port --all
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunRemovePort(f, cobraCmd, args)
		}}

	portCmd.Flags().StringVar(&cmd.LabelSelector, "label-selector", "", "Comma separated key=value selector list (e.g. release=test)")
	portCmd.Flags().BoolVar(&cmd.RemoveAll, "all", false, "Remove all configured ports")

	return portCmd
}

// RunRemovePort executes the remove port command logic
func (cmd *portCmd) RunRemovePort(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
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
	configInterface, err := configLoader.Load(cmd.ToConfigOptions(), log)
	if err != nil {
		return err
	}

	config := configInterface.Config()
	configureManager := f.NewConfigureManager(config, log)
	err = configureManager.RemovePort(cmd.RemoveAll, cmd.LabelSelector, args)
	if err != nil {
		return err
	}

	err = configLoader.Save(config)
	if err != nil {
		return err
	}

	log.Done("Successfully removed port")
	return nil
}
