package set

import (
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewSetCmd creates a new cobra command for the use sub command
func NewSetCmd(f factory.Factory) *cobra.Command {
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Make global configuration changes",
		Long: `
#######################################################
#################### devspace set #####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	setCmd.AddCommand(newAnalyticsCmd(f))
	setCmd.AddCommand(newVarCmd(f))
	setCmd.AddCommand(newEncryptionKeyCmd(f))

	return setCmd
}
