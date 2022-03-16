package reset

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewResetCmd creates a new cobra command
func NewResetCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
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

	resetCmd.AddCommand(newVarsCmd(f, globalFlags))
	resetCmd.AddCommand(newDependenciesCmd(f))
	resetCmd.AddCommand(newPodsCmd(f, globalFlags))

	// Add plugin commands
	plugin.AddPluginCommands(resetCmd, plugins, "reset")
	return resetCmd
}
