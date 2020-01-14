package list

import (
	"github.com/devspace-cloud/devspace/pkg/util/factory"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type SpacesCmd struct {
	Name     string
	Provider string
	All      bool
	Cluster  string
}

func newSpacesCmd(f factory.Factory) *cobra.Command {
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunListSpaces(f, cobraCmd, args)
		}}

	SpacesCmd.Flags().StringVar(&cmd.Name, "name", "", "Space name to show (default: all)")
	SpacesCmd.Flags().StringVar(&cmd.Provider, "provider", "", "Cloud Provider to use")
	SpacesCmd.Flags().StringVar(&cmd.Cluster, "cluster", "", "List all spaces in a certain cluster")
	SpacesCmd.Flags().BoolVar(&cmd.All, "all", false, "List all spaces the user has access to in all clusters (not only created by the user)")

	return SpacesCmd
}

// RunListCloudDevspaces executes the "devspace list spaces" functionality
func (cmd *SpacesCmd) RunListSpaces(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	logger := f.GetLog()
	// Get provider
	provider, err := f.GetProvider(cmd.Provider, logger)
	if err != nil {
		return errors.Wrap(err, "log into provider")
	}

	err = provider.PrintSpaces(cmd.Cluster, cmd.Name, cmd.All)
	if err != nil {
		return err
	}

	return nil
}
