package remove

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewRemoveCmd creates a new cobra command
func NewRemoveCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
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

	removeCmd.AddCommand(newClusterCmd(f))
	removeCmd.AddCommand(newContextCmd(f))
	removeCmd.AddCommand(newDeploymentCmd(globalFlags))
	removeCmd.AddCommand(newImageCmd(globalFlags))
	removeCmd.AddCommand(newPortCmd(globalFlags))
	removeCmd.AddCommand(newProviderCmd())
	removeCmd.AddCommand(newSpaceCmd(f))
	removeCmd.AddCommand(newSyncCmd(globalFlags))

	return removeCmd
}
