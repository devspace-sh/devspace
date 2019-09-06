package list

import (
	"github.com/spf13/cobra"
)

// NewListCmd creates a new cobra command
func NewListCmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists configuration",
		Long: `
#######################################################
#################### devspace list ####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	listCmd.AddCommand(newSyncCmd())
	listCmd.AddCommand(newSpacesCmd())
	listCmd.AddCommand(newClustersCmd())
	listCmd.AddCommand(newSelectorsCmd())
	listCmd.AddCommand(newPortsCmd())
	listCmd.AddCommand(newProfilesCmd())
	listCmd.AddCommand(newVarsCmd())
	listCmd.AddCommand(newDeploymentsCmd())
	listCmd.AddCommand(newProvidersCmd())
	listCmd.AddCommand(newAvailableComponentsCmd())

	return listCmd
}
