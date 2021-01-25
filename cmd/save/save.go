package save

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewSaveCmd creates a new cobra command for the sub command
func NewSaveCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save",
		Short: "Save configuration",
		Long: `
#######################################################
#################### devspace save ####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	cmd.AddCommand(newVarsCmd(f, globalFlags))

	// Add plugin commands
	plugin.AddPluginCommands(cmd, plugins, "save")
	return cmd
}
