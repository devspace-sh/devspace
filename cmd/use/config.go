package use

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configs"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type configCmd struct{}

func newConfigCmd() *cobra.Command {
	cmd := &configCmd{}

	return &cobra.Command{
		Use:   "config",
		Short: "Use a specific DevSpace configuration",
		Long: `
#######################################################
################ devspace use config ##################
#######################################################
Use a specific DevSpace configuration that is defined
in .devspace/configs.yaml

Example:
devspace use config myconfig
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunUseConfig,
	}
}

// RunUseConfig executes the "devspace use config command" logic
func (*configCmd) RunUseConfig(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	configs := configs.Configs{}
	err = configutil.LoadConfigs(&configs, configutil.DefaultConfigsPath)
	if err != nil {
		log.Fatalf("Cannot load %s: %v", configutil.DefaultConfigsPath, err)
	}

	// Check if config exists
	if _, ok := configs[args[0]]; ok == false {
		log.Fatalf("Config '%s' does not exist in %s", args[0], configutil.DefaultConfigsPath)
	}

	// Load generated config
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatalf("Cannot load generated config: %v", err)
	}

	// Exchange active config
	generatedConfig.ActiveConfig = args[0]

	// Save generated config
	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		log.Fatalf("Error saving generated config: %v", err)
	}

	log.Infof("Successfully switched to config '%s'", args[0])
}
