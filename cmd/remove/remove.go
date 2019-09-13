package remove

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/spf13/cobra"
)

// NewRemoveCmd creates a new cobra command
func NewRemoveCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
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

	removeCmd.AddCommand(newClusterCmd())
	removeCmd.AddCommand(newContextCmd())
	removeCmd.AddCommand(newDeploymentCmd(globalFlags))
	removeCmd.AddCommand(newImageCmd(globalFlags))
	removeCmd.AddCommand(newPortCmd(globalFlags))
	removeCmd.AddCommand(newProviderCmd())
	removeCmd.AddCommand(newSpaceCmd())
	removeCmd.AddCommand(newSyncCmd(globalFlags))

	return removeCmd
}
