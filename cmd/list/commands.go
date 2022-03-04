package list

import (
	"context"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
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
	configLoader, err := f.NewConfigLoader("")
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(logger)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Parse commands
	commandsInterface, err := configLoader.LoadWithParser(context.Background(), nil, nil, loader.NewCommandsParser(), nil, logger)
	if err != nil {
		return err
	}
	commands := commandsInterface.Config().Commands

	// Specify the table column names
	headerColumnNames := []string{
		"Name",
		"Command",
		"Description",
	}

	rows := [][]string{}
	for _, command := range commands {
		if command.Internal {
			continue
		}

		rows = append(rows, []string{
			command.Name,
			command.Command,
			command.Description,
		})
	}

	log.PrintTable(logger, headerColumnNames, rows)
	return nil
}
