package status

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/plugin"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewStatusCmd creates a new cobra command for the status sub command
func NewStatusCmd(f factory.Factory, plugins []plugin.Metadata) *cobra.Command {
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show the current status",
		Long: `
#######################################################
################## devspace status ####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	statusCmd.AddCommand(newSyncCmd(f))

	// Add plugin commands
	plugin.AddPluginCommands(statusCmd, plugins, "status")
	return statusCmd
}
