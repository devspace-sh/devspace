package list

import (
	"fmt"
	"strings"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type varsCmd struct {
	*flags.GlobalFlags
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

	return varsCmd
}

// RunListVars runs the list vars command logic
func (cmd *varsCmd) RunListVars(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	logger := f.GetLog()
	// Set config root
	configLoader := f.NewConfigLoader(cmd.ToConfigOptions(), logger)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Fill variables config
	_, err = configLoader.Load()
	if err != nil {
		return err
	}

	// Load generated config
	generatedConfig, err := configLoader.Generated()
	if err != nil {
		return err
	}

	// Specify the table column names
	headerColumnNames := []string{
		"Variable",
		"Value",
	}

	varRow := make([][]string, 0, len(generatedConfig.Vars))
	for name, value := range generatedConfig.Vars {
		if strings.HasPrefix(name, "DEVSPACE_SPACE_DOMAIN") {
			continue
		}

		varRow = append(varRow, []string{
			name,
			fmt.Sprintf("%v", value),
		})
	}

	// No variable found
	if len(varRow) == 0 {
		logger.Info("No variables found")
		return nil
	}

	log.PrintTable(logger, headerColumnNames, varRow)
	return nil
}
