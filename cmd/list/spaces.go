package list

import (
	cloudpkg "github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/util/log"
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
	############ devspace list cloud spaces ###############
	#######################################################
	List all cloud spaces

	Example:
	devspace list spaces
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunListCloudDevspaces,
	}

	spacesCmd.Flags().StringVar(&cmd.Name, "name", "", "Space name to show (default: all)")
	return spacesCmd
}

// RunListCloudDevspaces executes the devspace list cloud devspaces functionality
func (cmd *spacesCmd) RunListCloudDevspaces(cobraCmd *cobra.Command, args []string) {
	provider, err := cloudpkg.GetCurrentProvider(log.GetInstance())
	if err != nil {
		log.Fatalf("Error getting cloud context: %v", err)
	}
	if provider == nil {
		log.Fatal("No cloud provider specified")
	}

	err = provider.PrintSpaces(cmd.Name)
	if err != nil {
		log.Fatal(err)
	}
}
