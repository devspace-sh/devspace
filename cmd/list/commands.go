package list

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

type commandsCmd struct {
	*flags.GlobalFlags
}

func newCommandsCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunListProfiles(f, cobraCmd, args)
		}}

	return commandsCmd
}

// RunListCommands runs the list  command logic
func (cmd *commandsCmd) RunListProfiles(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	logger := f.GetLog()
	// Set config root
	configLoader := f.NewConfigLoader(nil, logger)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Parse commands
	commands, err := configLoader.ParseCommands()
	if err != nil {
		return err
	}

	// Save variables
	err = configLoader.SaveGenerated()
	if err != nil {
		return err
	}

	// Specify the table column names
	headerColumnNames := []string{
		"Name",
		"Command",
		"Description",
	}

	rows := [][]string{}
	for _, command := range commands {
		rows = append(rows, []string{
			command.Name,
			command.Command,
			command.Description,
		})
	}

	log.PrintTable(logger, headerColumnNames, rows)
	return nil
}
