package reset

import (
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/message"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type varsCmd struct{}

func newVarsCmd(f factory.Factory) *cobra.Command {
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunResetVars(f, cobraCmd, args)
		}}

	return varsCmd
}

// RunResetVars executes the reset vars command logic
func (cmd *varsCmd) RunResetVars(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	configLoader := f.NewConfigLoader("")
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Load generated config
	generatedConfig, err := configLoader.LoadGenerated(nil)
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

	log.Donef("Successfully deleted all variables")
	return nil
}
