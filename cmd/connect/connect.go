package connect

import (
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewConnectCmd creates a new cobra command
func NewConnectCmd(f factory.Factory) *cobra.Command {
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

	connectCmd.AddCommand(newClusterCmd(f))

	return connectCmd
}
