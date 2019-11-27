package list

import (
	"fmt"
	"strings"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type varsCmd struct {
	*flags.GlobalFlags
}

func newVarsCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
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
		RunE: cmd.RunListVars,
	}

	return varsCmd
}

// RunListVars runs the list vars command logic
func (cmd *varsCmd) RunListVars(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configLoader := loader.NewConfigLoader(cmd.ToConfigOptions(), log.GetInstance())
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
		log.GetInstance().Info("No variables found")
		return nil
	}

	log.PrintTable(log.GetInstance(), headerColumnNames, varRow)
	return nil
}
