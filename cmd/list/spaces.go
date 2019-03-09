package list

import (
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type spacesCmd struct {
	Name     string
	Provider string
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
	spacesCmd.Flags().StringVar(&cmd.Provider, "provider", "", "Cloud Provider to use")

	return spacesCmd
}

// RunListCloudDevspaces executes the "devspace list spaces" functionality
func (cmd *spacesCmd) RunListSpaces(cobraCmd *cobra.Command, args []string) {
	// Check if user has specified a certain provider
	var cloudProvider *string
	if cmd.Provider != "" {
		cloudProvider = &cmd.Provider
	}

	// Get provider
	provider, err := cloudpkg.GetProvider(cloudProvider, log.GetInstance())
	if err != nil {
		log.Fatalf("Error getting cloud context: %v", err)
	}

	err = provider.PrintSpaces(cmd.Name)
	if err != nil {
		log.Fatal(err)
	}
}
