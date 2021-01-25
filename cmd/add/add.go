package add

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewAddCmd creates a new cobra command
func NewAddCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
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
	addCmd.AddCommand(newPortCmd(f, globalFlags))
	addCmd.AddCommand(newImageCmd(f, globalFlags))
	addCmd.AddCommand(newDeploymentCmd(f, globalFlags))
	addCmd.AddCommand(newPluginCmd(f))

	// Add plugin commands
	plugin.AddPluginCommands(addCmd, plugins, "add")
	return addCmd
}
