package reset

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/spf13/cobra"
)

type varsCmd struct{}

func newVarsCmd() *cobra.Command {
	cmd := &varsCmd{}

	varsCmd := &cobra.Command{
		Use:   "vars",
		Short: "Resets the current config vars",
		Long: `
#######################################################
############### devspace reset vars ###################
#######################################################
Resets the saved variables of the current config

Examples:
devspace reset vars
#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunResetVars,
	}

	return varsCmd
}

// RunResetVars executes the reset vars command logic
func (cmd *varsCmd) RunResetVars(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Load generated config
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading generated.yaml: %v", err)
	}

	// Clear the vars map
	generatedConfig.Vars = map[string]string{}

	// Save the config
	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		log.Fatalf("Error saving config: %v", err)
	}

	log.Donef("Successfully deleted all variables")
}
