package update

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/spf13/cobra"
)

// NewUpdateCmd creates a new cobra command for the status sub command
func NewUpdateCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Updates the current config",
		Long: `
#######################################################
################## devspace update ####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	updateCmd.AddCommand(newConfigCmd())
	updateCmd.AddCommand(newDependenciesCmd(globalFlags))

	return updateCmd
}
