package list

import (
	"fmt"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/log"
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
		Run:  cmd.RunListVars,
	}

	return varsCmd
}

// RunListVars runs the list vars command logic
func (cmd *varsCmd) RunListVars(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	// Fill variables config
	configutil.GetConfig(cmd.KubeContext, cmd.Profile)

	// Load generated config
	generatedConfig, err := generated.LoadConfig(cmd.KubeContext)
	if err != nil {
		log.Fatal(err)
	}

	// No variable found
	if generatedConfig.Vars == nil || len(generatedConfig.Vars) == 0 {
		log.Info("No variables found")
		return
	}

	// Specify the table column names
	headerColumnNames := []string{
		"Variable",
		"Value",
	}

	varRow := make([][]string, 0, len(generatedConfig.Vars))

	for name, value := range generatedConfig.Vars {
		varRow = append(varRow, []string{
			name,
			fmt.Sprintf("%v", value),
		})
	}

	log.PrintTable(log.GetInstance(), headerColumnNames, varRow)
}
