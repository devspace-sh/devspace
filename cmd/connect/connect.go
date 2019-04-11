package connect

import "github.com/spf13/cobra"

// NewConnectCmd creates a new cobra command
func NewConnectCmd() *cobra.Command {
	connectCmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect an external cluster to devspace cloud",
		Long: `
#######################################################
################# devspace connect ####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	return connectCmd
}
