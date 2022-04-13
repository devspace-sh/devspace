package set

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewSetCmd creates a new cobra command for the use sub command
func NewSetCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Sets global configuration changes",
		Long: `
#######################################################
#################### devspace set #####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	setCmd.AddCommand(newVarCmd(f, globalFlags))

	// Add plugin commands
	plugin.AddPluginCommands(setCmd, plugins, "set")
	return setCmd
}
