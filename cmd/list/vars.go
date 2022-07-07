package list

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type varsCmd struct {
	*flags.GlobalFlags

	Output string
}

func newVarsCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &varsCmd{GlobalFlags: globalFlags}

	varsCmd := &cobra.Command{
		Use:   "vars",
		Short: "Lists the vars in the active config",
		Long: `
#######################################################
############### devspace list vars ####################
#######################################################
Lists the defined vars in the devspace config with their
values
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunListVars(f, cobraCmd, args)
		}}

	varsCmd.Flags().StringVarP(&cmd.Output, "output", "o", "", "The output format of the command. Can be either empty, keyvalue or json")
	return varsCmd
}

// RunListVars runs the list vars command logic
func (cmd *varsCmd) RunListVars(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	logger := f.GetLog()
	// Set config root
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
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

	// Fill variables config
	config, err := configLoader.Load(context.Background(), nil, cmd.ToConfigOptions(), logger)
	if err != nil {
		return err
	}

	switch cmd.Output {
	case "":
		// Specify the table column names
		headerColumnNames := []string{
			"Variable",
			"Value",
		}

		varRow := make([][]string, 0, len(config.Variables()))
		for name, value := range config.Variables() {
			varRow = append(varRow, []string{
				name,
				fmt.Sprintf("%v", value),
			})
		}
		sort.Slice(varRow, func(i, j int) bool {
			return varRow[i][0] < varRow[j][0]
		})

		// No variable found
		if len(varRow) == 0 {
			logger.Info("No variables found")
			return nil
		}

		sort.SliceStable(varRow, func(i, j int) bool {
			return varRow[i][0] < varRow[j][0]
		})

		log.PrintTable(logger, headerColumnNames, varRow)
	case "keyvalue":
		for name, value := range config.Variables() {
			fmt.Printf("%s=%v\n", name, value)
		}
	case "json":
		out, err := json.MarshalIndent(config.Variables(), "", "  ")
		if err != nil {
			return err
		}

		fmt.Print(string(out))
	default:
		return errors.Errorf("unsupported value for flag --output: %s", cmd.Output)
	}

	return nil
}
