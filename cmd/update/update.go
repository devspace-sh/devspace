package update

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewUpdateCmd creates a new cobra command for the status sub command
func NewUpdateCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
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

	updateCmd.AddCommand(newConfigCmd(f, globalFlags))
	updateCmd.AddCommand(newDependenciesCmd(f, globalFlags))

	return updateCmd
}
