package use

import (
	"github.com/spf13/cobra"
)

// NewUseCmd creates a new cobra command for the use sub command
func NewUseCmd() *cobra.Command {
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

	useCmd.AddCommand(newConfigCmd())
	useCmd.AddCommand(newContextCmd())
	useCmd.AddCommand(newNamespaceCmd())
	useCmd.AddCommand(newProviderCmd())
	useCmd.AddCommand(newSpaceCmd())

	return useCmd
}
