package list

import (
	"context"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"sort"

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
			return cmd.RunListCommands(f, cobraCmd, args)
		}}

	return commandsCmd
}

// RunListCommands runs the list command logic
func (cmd *commandsCmd) RunListCommands(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
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
		"Section",
		"Name",
		"Description",
	}

	sections := map[string][][]string{}
	for _, command := range commands {
		if command.Internal {
			continue
		}

		sections[command.Section] = append(sections[command.Section], []string{
			command.Name,
			command.Description,
		})
	}

	allRows := [][]string{}
	for section, r := range sections {
		sort.Slice(r, func(i, j int) bool {
			return r[i][0] < r[j][0]
		})
		if section == "" && len(sections) == 1 {
			headerColumnNames = []string{"Name", "Description"}
			allRows = r
			break
		}

		for _, ri := range r {
			allRows = append(allRows, []string{
				section,
				ri[0],
				ri[1],
			})
		}
	}
	sort.SliceStable(allRows, func(i, j int) bool {
		return allRows[i][0] < allRows[j][0]
	})
	log.PrintTableWithOptions(logger, headerColumnNames, allRows, func(table *tablewriter.Table) {
		table.SetAutoMergeCells(true)
	})
	return nil
}
