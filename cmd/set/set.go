package set

import (
	"github.com/spf13/cobra"
)

// NewSetCmd creates a new cobra command for the use sub command
func NewSetCmd() *cobra.Command {
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Make global configuration changes",
		Long: `
#######################################################
################# devspace set ##################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	setCmd.AddCommand(newAnalyticsCmd())

	return setCmd
}
