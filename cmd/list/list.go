package list

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/spf13/cobra"
)

// NewListCmd creates a new cobra command
func NewListCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
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

	listCmd.AddCommand(newSyncCmd(globalFlags))
	listCmd.AddCommand(newSpacesCmd())
	listCmd.AddCommand(newClustersCmd())
	listCmd.AddCommand(newPortsCmd(globalFlags))
	listCmd.AddCommand(newProfilesCmd())
	listCmd.AddCommand(newVarsCmd(globalFlags))
	listCmd.AddCommand(newDeploymentsCmd(globalFlags))
	listCmd.AddCommand(newProvidersCmd())
	listCmd.AddCommand(newAvailableComponentsCmd())
	listCmd.AddCommand(newCommandsCmd(globalFlags))

	return listCmd
}
