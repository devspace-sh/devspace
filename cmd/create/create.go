package create

import (
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

// NewCreateCmd creates a new cobra command
func NewCreateCmd(f factory.Factory) *cobra.Command {
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

	createCmd.AddCommand(newSpaceCmd(f))

	return createCmd
}
