package reset

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/message"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type varsCmd struct {
	*flags.GlobalFlags
}

func newVarsCmd(f factory.Factory, flags *flags.GlobalFlags) *cobra.Command {
	cmd := &varsCmd{
		GlobalFlags: flags,
	}

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
			return cmd.RunResetVars(f)
		}}

	return varsCmd
}

// RunResetVars executes the reset vars command logic
func (cmd *varsCmd) RunResetVars(f factory.Factory) error {
	// Set config root
	log := f.GetLog()
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Load generated config
	localCache, err := configLoader.LoadLocalCache()
	if err != nil {
		return err
	}

	// Clear the vars map
	localCache.ClearVars()

	// Save the config
	err = localCache.Save()
	if err != nil {
		return errors.Errorf("Error saving config: %v", err)
	}

	log.Donef("Successfully deleted all variables")
	return nil
}
