package set

import (
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewSetCmd creates a new cobra command for the use sub command
func NewSetCmd(f factory.Factory, plugins []plugin.Metadata) *cobra.Command {
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Make global configuration changes",
		Long: `
#######################################################
#################### devspace set #####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	setCmd.AddCommand(newVarCmd(f))

	// Add plugin commands
	plugin.AddPluginCommands(setCmd, plugins, "set")
	return setCmd
}
