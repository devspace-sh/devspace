package update

import (
	"context"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// configCmd holds the cmd flags
type configCmd struct{}

// newConfigCmd creates a new command
func newConfigCmd() *cobra.Command {
	cmd := &configCmd{}

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
		Run:  cmd.RunConfig,
	}

	return configCmd
}

// RunConfig executes the functionality "devspace update config"
func (cmd *configCmd) RunConfig(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Get config
	configutil.GetBaseConfig(context.Background())

	// Save it
	err = configutil.SaveLoadedConfig()
	if err != nil {
		log.Fatalf("Error saving config: %v", err)
	}

	log.Infof("Successfully converted base config to current version")
}
