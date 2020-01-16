package add

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewAddCmd creates a new cobra command
func NewAddCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
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

	addCmd.AddCommand(newSyncCmd(f, globalFlags))
	addCmd.AddCommand(newProviderCmd(f))
	addCmd.AddCommand(newPortCmd(f, globalFlags))
	addCmd.AddCommand(newImageCmd(f, globalFlags))
	addCmd.AddCommand(newDeploymentCmd(f, globalFlags))

	return addCmd
}
