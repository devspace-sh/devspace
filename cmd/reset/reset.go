package reset

import (
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewResetCmd creates a new cobra command
func NewResetCmd(f factory.Factory) *cobra.Command {
	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Resets an cluster token",
		Long: `
#######################################################
################## devspace reset #####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	resetCmd.AddCommand(newKeyCmd(f))
	resetCmd.AddCommand(newVarsCmd(f))
	resetCmd.AddCommand(newDependenciesCmd(f))

	return resetCmd
}
