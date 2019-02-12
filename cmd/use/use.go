package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	v1 "github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// UseCmd holds the information needed for the use command
type UseCmd struct {
	flags *UseCmdFlags
}

// UseCmdFlags holds the possible flags for the use command
type UseCmdFlags struct {
}

func init() {
	cmd := &UseCmd{
		flags: &UseCmdFlags{},
	}

	useCmd := &cobra.Command{
		Use:   "use",
		Short: "Use specific config",
		Long: `
	#######################################################
	#################### devspace use #####################
	#######################################################
	`,
		Args: cobra.NoArgs,
	}

	rootCmd.AddCommand(useCmd)

	useConfigCmd := &cobra.Command{
		Use:   "config",
		Short: "Use a specific devspace configuration",
		Long: `
	#######################################################
	################ devspace use config ##################
	#######################################################
	Use a specific devspace configuration that is defined
	in .devspace/configs.yaml

	Example:
	devspace use config myconfig
	#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunUseConfig,
	}

	useCmd.AddCommand(useConfigCmd)
}

// RunUseConfig executes the devspace use config command logic
func (*UseCmd) RunUseConfig(cobraCmd *cobra.Command, args []string) {
	configs := v1.Configs{}
	err := configutil.LoadConfigs(&configs, configutil.DefaultConfigsPath)
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
	generatedConfig.ActiveConfig = &args[0]

	// Save generated config
	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		log.Fatalf("Error saving generated config: %v", err)
	}

	log.Infof("Successfully switched to config '%s'", args[0])
}
