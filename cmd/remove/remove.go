package remove

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewRemoveCmd creates a new cobra command
func NewRemoveCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
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

	removeCmd.AddCommand(newContextCmd(f))
	removeCmd.AddCommand(newDeploymentCmd(f, globalFlags))
	removeCmd.AddCommand(newImageCmd(f, globalFlags))
	removeCmd.AddCommand(newPortCmd(f, globalFlags))
	removeCmd.AddCommand(newPluginCmd(f))
	removeCmd.AddCommand(newSyncCmd(f, globalFlags))

	// Add plugin commands
	plugin.AddPluginCommands(removeCmd, plugins, "remove")
	return removeCmd
}
