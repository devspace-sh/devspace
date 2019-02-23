package update

import (
	"github.com/spf13/cobra"
)

// NewUpdateCmd creates a new cobra command for the status sub command
func NewUpdateCmd() *cobra.Command {
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

	return updateCmd
}
