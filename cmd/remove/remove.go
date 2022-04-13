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
		Short: "Removes devspace configuration",
		Long: `
#######################################################
################## devspace remove ####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	removeCmd.AddCommand(newContextCmd(f))
	removeCmd.AddCommand(newPluginCmd(f))

	// Add plugin commands
	plugin.AddPluginCommands(removeCmd, plugins, "remove")
	return removeCmd
}
