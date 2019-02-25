package list

import (
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type spacesCmd struct {
	Name string
}

func newSpacesCmd() *cobra.Command {
	cmd := &spacesCmd{}

	spacesCmd := &cobra.Command{
		Use:   "spaces",
		Short: "Lists all user spaces",
		Long: `
#######################################################
############### devspace list spaces ##################
#######################################################
List all cloud spaces

Example:
devspace list spaces
#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunListSpaces,
	}

	spacesCmd.Flags().StringVar(&cmd.Name, "name", "", "Space name to show (default: all)")
	return spacesCmd
}

// RunListCloudDevspaces executes the "devspace list spaces" functionality
func (cmd *spacesCmd) RunListSpaces(cobraCmd *cobra.Command, args []string) {
	provider, err := cloudpkg.GetCurrentProvider(log.GetInstance())
	if err != nil {
		log.Fatalf("Error getting cloud context: %v", err)
	}

	err = provider.PrintSpaces(cmd.Name)
	if err != nil {
		log.Fatal(err)
	}
}
