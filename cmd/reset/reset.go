package reset

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/plugin"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewResetCmd creates a new cobra command
func NewResetCmd(f factory.Factory, plugins []plugin.Metadata) *cobra.Command {
	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Resets an cluster token",
		Long: `
#######################################################
################## devspace reset #####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	resetCmd.AddCommand(newKeyCmd(f))
	resetCmd.AddCommand(newVarsCmd(f))
	resetCmd.AddCommand(newDependenciesCmd(f))

	// Add plugin commands
	plugin.AddPluginCommands(resetCmd, plugins, "reset")
	return resetCmd
}
