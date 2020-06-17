package sync

import (
	"github.com/spf13/cobra"
)

// NewSyncCmd creates a new cobra command
func NewSyncCmd() *cobra.Command {
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync holds the sync relevant commands",
		Args:  cobra.NoArgs,
	}

	syncCmd.AddCommand(NewDownstreamCmd())
	syncCmd.AddCommand(NewUpstreamCmd())
	return syncCmd
}
