package add

import (
	"github.com/spf13/cobra"
)

// NewAddCmd creates a new cobra command
func NewAddCmd() *cobra.Command {
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Change the devspace configuration",
		Long: `
	#######################################################
	#################### devspace add #####################
	#######################################################
	`,
		Args: cobra.NoArgs,
	}

	addCmd.AddCommand(newSyncCmd())
	addCmd.AddCommand(newServiceCmd())
	addCmd.AddCommand(newProviderCmd())
	addCmd.AddCommand(newPortCmd())
	addCmd.AddCommand(newPackageCmd())
	addCmd.AddCommand(newImageCmd())
	addCmd.AddCommand(newDeploymentCmd())

	return addCmd
}
