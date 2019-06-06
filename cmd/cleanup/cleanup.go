package cleanup

import (
	"github.com/spf13/cobra"
)

// NewCleanupCmd creates a new cobra command
func NewCleanupCmd() *cobra.Command {
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

	cleanupCmd.AddCommand(newImagesCmd())

	return cleanupCmd
}
