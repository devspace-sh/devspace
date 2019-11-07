package add

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/spf13/cobra"
)

// NewAddCmd creates a new cobra command
func NewAddCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Convenience command: adds something to devspace.yaml",
		Long: `
#######################################################
#################### devspace add #####################
#######################################################
Adds config sections to devspace.yaml
	`,
		Args: cobra.NoArgs,
	}

	addCmd.AddCommand(newSyncCmd(globalFlags))
	addCmd.AddCommand(newProviderCmd())
	addCmd.AddCommand(newPortCmd(globalFlags))
	addCmd.AddCommand(newImageCmd(globalFlags))
	addCmd.AddCommand(newDeploymentCmd(globalFlags))

	return addCmd
}
