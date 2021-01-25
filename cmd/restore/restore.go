package restore

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewRestoreCmd creates a new cobra command for the sub command
func NewRestoreCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore configuration",
		Long: `
#######################################################
################## devspace restore ###################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	cmd.AddCommand(newVarsCmd(f, globalFlags))

	// Add plugin commands
	plugin.AddPluginCommands(cmd, plugins, "restore")
	return cmd
}
