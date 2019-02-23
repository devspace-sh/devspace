package create

import (
	"github.com/spf13/cobra"
)

// NewCreateCmd creates a new cobra command
func NewCreateCmd() *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create spaces in the cloud",
		Long: `
#######################################################
################## devspace create ####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	createCmd.AddCommand(newSpaceCmd())

	return createCmd
}
