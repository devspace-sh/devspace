package cmd

import "github.com/spf13/cobra"

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
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunUseConfig,
	}

	useCmd.AddCommand(useConfigCmd)
}

// RunUseConfig executes the devspace use config command logic
func (*UseCmd) RunUseConfig(cobraCmd *cobra.Command, args []string) {

}
