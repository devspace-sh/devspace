package update

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// configCmd holds the cmd flags
type configCmd struct {
	*flags.GlobalFlags
}

// newConfigCmd creates a new command
func newConfigCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
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
		RunE: cmd.RunConfig,
	}

	return configCmd
}

// RunConfig executes the functionality "devspace update config"
func (cmd *configCmd) RunConfig(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := log.GetInstance()
	configLoader := loader.NewConfigLoader(cmd.ToConfigOptions(), log)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	} else if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Get profiles
	profiles, err := loader.GetProfiles(".")
	if err != nil {
		return err
	}

	// Get config
	config, err := configLoader.LoadWithoutProfile()
	if err != nil {
		return errors.Wrap(err, "load config")
	}

	// Save it
	err = configLoader.Save(config)
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
