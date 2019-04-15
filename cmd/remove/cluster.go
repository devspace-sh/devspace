package remove

import (
	"fmt"

	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/spf13/cobra"
)

type clusterCmd struct {
	Provider string
}

func newClusterCmd() *cobra.Command {
	cmd := &clusterCmd{}

	clusterCmd := &cobra.Command{
		Use:   "cluster",
		Short: "Removes a connected cluster",
		Long: `
#######################################################
############# devspace remove cluster #################
#######################################################
Removes a connected cluster 

Example:
devspace remove cluster my-cluster
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunRemoveCluster,
	}

	clusterCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")

	return clusterCmd
}

// RunRemoveCluster executes the devspace remove cluster functionality
func (cmd *clusterCmd) RunRemoveCluster(cobraCmd *cobra.Command, args []string) {
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

	// Verify user is sure to delete the cluster
	deleteCluster := survey.Question(&survey.QuestionOptions{
		Question:     fmt.Sprintf("Are you sure you want to delete cluster %s? This action is irreversible", args[0]),
		DefaultValue: "No",
		Options: []string{
			"No",
			"Yes",
		},
	}) == "Yes"
	if deleteCluster == false {
		return
	}

	// Get cluster by name
	cluster, err := provider.GetClusterByName(args[0])
	if err != nil {
		log.Fatal(err)
	}

	// Delete all spaces?
	deleteSpaces := survey.Question(&survey.QuestionOptions{
		Question:     "Do you want to delete all cluster spaces?",
		DefaultValue: "Yes",
		Options: []string{
			"Yes",
			"No",
		},
	}) == "Yes"

	// Delete services
	deleteServices := survey.Question(&survey.QuestionOptions{
		Question:     "Do you want to delete all cluster services?",
		DefaultValue: "Yes",
		Options: []string{
			"Yes",
			"No",
		},
	}) == "Yes"

	// Delete cluster
	log.StartWait("Deleting cluster " + cluster.Name)
	err = provider.DeleteCluster(cluster, deleteServices, deleteSpaces)
	if err != nil {
		log.Fatal(err)
	}
	log.StopWait()

	delete(provider.ClusterKey, cluster.ClusterID)
	err = provider.Save()
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully deleted cluster %s", args[0])
}
