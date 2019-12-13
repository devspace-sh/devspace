package reset

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"

	"github.com/pkg/errors"
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
		RunE: cmd.RunResetVars,
	}

	return varsCmd
}

// RunResetVars executes the reset vars command logic
func (cmd *varsCmd) RunResetVars(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configLoader := loader.NewConfigLoader(nil, log.GetInstance())
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Load generated config
	generatedConfig, err := configLoader.Generated()
	if err != nil {
		return err
	}

	// Clear the vars map
	generatedConfig.Vars = map[string]string{}

	// Save the config
	err = configLoader.SaveGenerated(generatedConfig)
	if err != nil {
		return errors.Errorf("Error saving config: %v", err)
	}

	log.GetInstance().Donef("Successfully deleted all variables")
	return nil
}
