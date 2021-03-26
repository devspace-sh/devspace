package update

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// configCmd holds the cmd flags
type configCmd struct {
	*flags.GlobalFlags
}

// newConfigCmd creates a new command
func newConfigCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &configCmd{GlobalFlags: globalFlags}

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Converts the active config to the current config version",
		Long: `
#######################################################
############### devspace update config ################
#######################################################
Updates the currently active config to the newest
config version

Note: This does not upgrade the overwrite configs
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunConfig(f, cobraCmd, args)
		}}

	return configCmd
}

// RunConfig executes the functionality "devspace update config"
func (cmd *configCmd) RunConfig(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	} else if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	log.Warn("This command is deprecated and will be removed in a future DevSpace version. Please modify the devspace.yaml directly instead")

	// Get config
	config, err := configLoader.Load(cmd.ToConfigOptions(), log)
	if err != nil {
		return errors.Wrap(err, "load config")
	}

	// Get profiles
	profiles, err := config.Profiles()
	if err != nil {
		return err
	}

	// Save it
	err = configLoader.Save(config.Config())
	if err != nil {
		return errors.Errorf("Error saving config: %v", err)
	}

	// Check if there are any profile patches
	if len(profiles) > 0 {
		log.Warnf("'devspace update config' does NOT update profiles[*].replace or profiles[*].patches. Please manually update any profiles[*].replace and profiles[*].patches")
	}

	log.Infof("Successfully converted base config to current version")
	return nil
}
