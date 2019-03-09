package remove

import (
	"github.com/spf13/cobra"
)

// NewRemoveCmd creates a new cobra command
func NewRemoveCmd() *cobra.Command {
	removeCmd := &cobra.Command{
		Use:   "remove",
		Short: "Changes devspace configuration",
		Long: `
#######################################################
################## devspace remove ####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	removeCmd.AddCommand(newDeploymentCmd())
	removeCmd.AddCommand(newImageCmd())
	removeCmd.AddCommand(newPackageCmd())
	removeCmd.AddCommand(newPortCmd())
	removeCmd.AddCommand(newProviderCmd())
	removeCmd.AddCommand(newSelectorCmd())
	removeCmd.AddCommand(newSpaceCmd())
	removeCmd.AddCommand(newSyncCmd())

	return removeCmd
}
