package cleanup

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/plugin"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewCleanupCmd creates a new cobra command
func NewCleanupCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
	cleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleans up resources",
		Long: `
#######################################################
################## devspace cleanup ###################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	cleanupCmd.AddCommand(newImagesCmd(f, globalFlags))

	// Add plugin commands
	plugin.AddPluginCommands(cleanupCmd, plugins, "cleanup")
	return cleanupCmd
}
