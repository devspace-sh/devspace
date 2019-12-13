package list

import (
	"strconv"

	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type clustersCmd struct {
	Provider string
	All      bool
}

func newClustersCmd() *cobra.Command {
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
		RunE: cmd.RunListClusters,
	}

	clustersCmd.Flags().StringVar(&cmd.Provider, "provider", "", "Cloud Provider to use")
	clustersCmd.Flags().BoolVar(&cmd.All, "all", false, "Show all available clusters including hosted DevSpace cloud clusters")

	return clustersCmd
}

// RunListClusters executes the "devspace list clusters" functionality
func (cmd *clustersCmd) RunListClusters(cobraCmd *cobra.Command, args []string) error {
	// Get provider
	log := logpkg.GetInstance()
	provider, err := cloudpkg.GetProvider(cmd.Provider, log)
	if err != nil {
		return errors.Wrap(err, "get provider")
	}

	log.StartWait("Retrieving clusters")
	clusters, err := provider.Client().GetClusters()
	if err != nil {
		return errors.Errorf("Error retrieving clusters: %v", err)
	}
	log.StopWait()

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
		logpkg.PrintTable(log, headerColumnNames, values)
	} else {
		log.Infof("No clusters found. You can connect a cluster with `%s`", ansi.Color("devspace connect cluster", "white+b"))
	}

	return nil
}
