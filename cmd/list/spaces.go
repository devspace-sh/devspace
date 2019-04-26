package list

import (
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type spacesCmd struct {
	Name     string
	Provider string
	All      bool
	Cluster  string
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
List all user cloud spaces

Example:
devspace list spaces
devspace list spaces --cluster my-cluster
devspace list spaces --all
#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunListSpaces,
	}

	spacesCmd.Flags().StringVar(&cmd.Name, "name", "", "Space name to show (default: all)")
	spacesCmd.Flags().StringVar(&cmd.Provider, "provider", "", "Cloud Provider to use")
	spacesCmd.Flags().StringVar(&cmd.Cluster, "cluster", "", "List all spaces in a certain cluster")
	spacesCmd.Flags().BoolVar(&cmd.All, "all", false, "List all spaces the user has access to in all clusters (not only created by the user)")

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

	err = provider.PrintSpaces(cmd.Cluster, cmd.Name, cmd.All)
	if err != nil {
		log.Fatal(err)
	}
}
