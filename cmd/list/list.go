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
	listCmd.AddCommand(newSelectorsCmd())
	listCmd.AddCommand(newPortsCmd())
	listCmd.AddCommand(newPackagesCmd())
	listCmd.AddCommand(newConfigsCmd())
	listCmd.AddCommand(newVarsCmd())
	listCmd.AddCommand(newProvidersCmd())

	return listCmd
}
