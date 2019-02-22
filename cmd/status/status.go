package status

import (
	"github.com/spf13/cobra"
)

// NewStatusCmd creates a new cobra command for the status sub command
func NewStatusCmd() *cobra.Command {
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

	statusCmd.AddCommand(newSyncCmd())
	statusCmd.AddCommand(newDeploymentsCmd())

	return statusCmd
}
