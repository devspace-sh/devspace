package list

import (
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/util/factory"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type clustersCmd struct {
	Provider string
	All      bool
}

func newClustersCmd(f factory.Factory) *cobra.Command {
	cmd := &clustersCmd{}

	clustersCmd := &cobra.Command{
		Use:   "clusters",
		Short: "Lists all connected clusters",
		Long: `
#######################################################
############## devspace list clusters #################
#######################################################
List all connected user clusters

Example:
devspace list clusters
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunListClusters(f, cobraCmd, args)
		}}

	clustersCmd.Flags().StringVar(&cmd.Provider, "provider", "", "Cloud Provider to use")
	clustersCmd.Flags().BoolVar(&cmd.All, "all", false, "Show all available clusters including hosted DevSpace cloud clusters")

	return clustersCmd
}

// RunListClusters executes the "devspace list clusters" functionality
func (cmd *clustersCmd) RunListClusters(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Get provider
	logger := f.GetLog()
	provider, err := f.GetProvider(cmd.Provider, logger)
	if err != nil {
		return errors.Wrap(err, "get provider")
	}

	logger.StartWait("Retrieving clusters")
	clusters, err := provider.Client().GetClusters()
	if err != nil {
		return errors.Errorf("Error retrieving clusters: %v", err)
	}
	logger.StopWait()

	headerColumnNames := []string{
		"ID",
		"Name",
		"Owner",
		"Created",
	}

	values := [][]string{}

	for _, cluster := range clusters {
		owner := ""
		createdAt := ""
		if cluster.Owner != nil {
			owner = cluster.Owner.Name

			if cluster.CreatedAt != nil {
				createdAt = *cluster.CreatedAt
			}
		} else if cmd.All == false {
			continue
		}

		values = append(values, []string{
			strconv.Itoa(cluster.ClusterID),
			cluster.Name,
			owner,
			createdAt,
		})
	}

	if len(values) > 0 {
		logpkg.PrintTable(logger, headerColumnNames, values)
	} else {
		logger.Infof("No clusters found. You can connect a cluster with `%s`", ansi.Color("devspace connect cluster", "white+b"))
	}

	return nil
}
