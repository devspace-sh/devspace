package list

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/plugin"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewListCmd creates a new cobra command
func NewListCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists configuration",
		Long: `
#######################################################
#################### devspace list ####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	listCmd.AddCommand(newSyncCmd(f, globalFlags))
	listCmd.AddCommand(newSpacesCmd(f))
	listCmd.AddCommand(newClustersCmd(f))
	listCmd.AddCommand(newPortsCmd(f, globalFlags))
	listCmd.AddCommand(newProfilesCmd(f))
	listCmd.AddCommand(newVarsCmd(f, globalFlags))
	listCmd.AddCommand(newDeploymentsCmd(f, globalFlags))
	listCmd.AddCommand(newProvidersCmd(f))
	listCmd.AddCommand(newContextsCmd(f))
	listCmd.AddCommand(newCommandsCmd(f, globalFlags))
	listCmd.AddCommand(newNamespacesCmd(f, globalFlags))

	// Add plugin commands
	plugin.AddPluginCommands(listCmd, plugins, "list")
	return listCmd
}
