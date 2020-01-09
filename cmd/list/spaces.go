package list

import (
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type SpacesCmd struct {
	Name     string
	Provider string
	All      bool
	Cluster  string
}

func newSpacesCmd() *cobra.Command {
	cmd := &SpacesCmd{}

	SpacesCmd := &cobra.Command{
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
		RunE: cmd.RunListSpaces,
	}

	SpacesCmd.Flags().StringVar(&cmd.Name, "name", "", "Space name to show (default: all)")
	SpacesCmd.Flags().StringVar(&cmd.Provider, "provider", "", "Cloud Provider to use")
	SpacesCmd.Flags().StringVar(&cmd.Cluster, "cluster", "", "List all spaces in a certain cluster")
	SpacesCmd.Flags().BoolVar(&cmd.All, "all", false, "List all spaces the user has access to in all clusters (not only created by the user)")

	return SpacesCmd
}

// RunListCloudDevspaces executes the "devspace list spaces" functionality
func (cmd *SpacesCmd) RunListSpaces(cobraCmd *cobra.Command, args []string) error {
	// Get provider
	provider, err := cloudpkg.GetProvider(cmd.Provider, log.GetInstance())
	if err != nil {
		return errors.Wrap(err, "log into provider")
	}

	err = provider.PrintSpaces(cmd.Cluster, cmd.Name, cmd.All)
	if err != nil {
		return err
	}

	return nil
}
