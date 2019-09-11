package add

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/spf13/cobra"
)

// NewAddCmd creates a new cobra command
func NewAddCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Change the DevSpace configuration",
		Long: `
#######################################################
#################### devspace add #####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	addCmd.AddCommand(newSyncCmd(globalFlags))
	addCmd.AddCommand(newProviderCmd())
	addCmd.AddCommand(newPortCmd(globalFlags))
	addCmd.AddCommand(newImageCmd())
	addCmd.AddCommand(newDeploymentCmd())

	return addCmd
}
