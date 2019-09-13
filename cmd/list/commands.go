package list

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

type commandsCmd struct {
	*flags.GlobalFlags
}

func newCommandsCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &commandsCmd{GlobalFlags: globalFlags}

	commandsCmd := &cobra.Command{
		Use:   "commands",
		Short: "Lists all custom DevSpace commands",
		Long: `
#######################################################
############## devspace list commands #################
#######################################################
Lists all DevSpace custom commands defined in the 
devspace.yaml
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: cmd.RunListProfiles,
	}

	return commandsCmd
}

// RunListCommands runs the list  command logic
func (cmd *commandsCmd) RunListProfiles(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Get config
	config, err := configutil.GetBaseConfig(configutil.FromFlags(cmd.GlobalFlags))
	if err != nil {
		return err
	}

	// Specify the table column names
	headerColumnNames := []string{
		"Name",
		"Command",
	}

	rows := [][]string{}
	for _, command := range config.Commands {
		rows = append(rows, []string{
			command.Name,
			command.Command,
		})
	}

	log.PrintTable(log.GetInstance(), headerColumnNames, rows)
	return nil
}
